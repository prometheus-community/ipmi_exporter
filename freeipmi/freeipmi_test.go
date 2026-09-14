// Copyright 2026 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package freeipmi

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

var testLogger = slog.New(slog.DiscardHandler)

// TestExecuteContextKillsHungCommand is the process-pileup regression test:
// a tool that outlives its deadline must be killed, together with any
// grandchildren in its process group, instead of blocking Wait forever
// (https://github.com/prometheus-community/ipmi_exporter/issues/95).
func TestExecuteContextKillsHungCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	// The backgrounded sleep inherits the output pipe; group-SIGTERM must
	// take down both processes.
	result := ExecuteContext(ctx, "/bin/sh", []string{"-c", "sleep 60 & exec sleep 60"}, "", "", testLogger)
	elapsed := time.Since(start)

	if result.err == nil {
		t.Fatalf("expected an error for a killed command, got none (output: %q)", result.output)
	}
	if !strings.Contains(result.err.Error(), "timed out") {
		t.Errorf("expected a timed-out error, got: %v", result.err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("command not killed at deadline: took %s", elapsed)
	}
}

// TestExecuteContextWaitDelayUnblocksOnHeldPipe covers the other hang: the
// child exits cleanly but a grandchild keeps the inherited stdout pipe open.
// WaitDelay must unblock Wait instead of waiting for the grandchild.
func TestExecuteContextWaitDelayUnblocksOnHeldPipe(t *testing.T) {
	oldDelay := waitDelay
	waitDelay = 300 * time.Millisecond
	defer func() { waitDelay = oldDelay }()

	start := time.Now()
	result := ExecuteContext(context.Background(), "/bin/sh", []string{"-c", "sleep 60 & exit 0"}, "", "", testLogger)
	elapsed := time.Since(start)

	// The child exited 0 but Wait was cut short while pipes were held: Go
	// reports exec.ErrWaitDelay, surfaced via the generic error path.
	if result.err == nil {
		t.Fatal("expected an error when output pipes are held past WaitDelay, got none")
	}
	if elapsed > 3*time.Second {
		t.Errorf("Wait not unblocked by WaitDelay: took %s", elapsed)
	}
}

// TestExecuteContextConfigArrivesOnFd3 proves the config reaches the child
// via the inherited pipe at /dev/fd/3, appended as --config-file.
func TestExecuteContextConfigArrivesOnFd3(t *testing.T) {
	config := "driver-type LAN\nprivilege-level admin\n"
	// argv after append: -c <script> sh --config-file /dev/fd/3
	// so $1 = --config-file, $2 = /dev/fd/3
	result := ExecuteContext(context.Background(), "/bin/sh", []string{"-c", `cat "$2"`, "sh"}, config, "", testLogger)

	if result.err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", result.err, result.output)
	}
	if string(result.output) != config {
		t.Errorf("config did not arrive via /dev/fd/3: got %q, want %q", result.output, config)
	}
}

// TestExecuteContextEmptyConfigStillPassed: an empty config must still be
// handed to the tool. The empty override deliberately masks any system-wide
// /etc/freeipmi/freeipmi.conf; dropping --config-file would unmask it and
// silently change behavior for local default modules.
func TestExecuteContextEmptyConfigStillPassed(t *testing.T) {
	result := ExecuteContext(context.Background(), "/bin/sh", []string{"-c", `echo "$#"`, "sh"}, "", "", testLogger)

	if result.err != nil {
		t.Fatalf("unexpected error: %v (output: %q)", result.err, result.output)
	}
	if got := strings.TrimSpace(string(result.output)); got != "2" {
		t.Errorf("expected 2 appended args (--config-file /dev/fd/3), $# = %s", got)
	}
}

// TestExecuteNoTimeoutRegression: the compatibility wrapper without a
// deadline must behave as before.
func TestExecuteNoTimeoutRegression(t *testing.T) {
	result := Execute("/bin/sh", []string{"-c", "echo hello"}, "", "", testLogger)

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if got := strings.TrimSpace(string(result.output)); got != "hello" {
		t.Errorf("expected output hello, got %q", got)
	}
}

// TestExecuteContextTargetAppended keeps the remote-target contract: -h
// must come after --config-file, matching the previous implementation.
func TestExecuteContextTargetAppended(t *testing.T) {
	result := ExecuteContext(context.Background(), "/bin/sh", []string{"-c", `echo "$3" "$4"`, "sh"}, "", "10.0.0.1", testLogger)

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if got := strings.TrimSpace(string(result.output)); got != "-h 10.0.0.1" {
		t.Errorf("expected trailing '-h 10.0.0.1', got %q", got)
	}
}
