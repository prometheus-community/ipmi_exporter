// Copyright 2021 The Prometheus Authors
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
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// waitDelay is how long Wait keeps waiting, after the child has exited or the
// context has been canceled, for the child's inherited output pipes to close
// before forcibly closing them (and, on cancellation, killing the process).
// This is the escape hatch for a grandchild that inherited stdout/stderr and
// outlives the child. A variable so tests can shorten it.
var waitDelay = 5 * time.Second

var (
	ipmiDCMIPowerMeasurementRegex       = regexp.MustCompile(`^Power Measurement\s*:\s*(?P<value>Active|Not\sAvailable).*`)
	ipmiDCMICurrentPowerRegex           = regexp.MustCompile(`^Current Power\s*:\s*(?P<value>[0-9.]*)\s*Watts.*`)
	ipmiChassisPowerRegex               = regexp.MustCompile(`^System Power\s*:\s(?P<value>.*)`)
	ipmiChassisDriveFaultRegex          = regexp.MustCompile(`^Drive Fault\s*:\s(?P<value>.*)`)
	ipmiChassisCoolingFaultRegex        = regexp.MustCompile(`^Cooling/fan fault\s*:\s(?P<value>.*)`)
	ipmiSELEntriesRegex                 = regexp.MustCompile(`^Number of log entries\s*:\s(?P<value>[0-9.]*)`)
	ipmiSELFreeSpaceRegex               = regexp.MustCompile(`^Free space remaining\s*:\s(?P<value>[0-9.]*)\s*bytes.*`)
	ipmiSELEventRegex                   = regexp.MustCompile(`^(?P<id>[0-9]+),\s*(?P<date>[^,]*),(?P<time>[^,]*),(?P<name>[^,]*),(?P<type>[^,]*),(?P<state>[^,]*),(?P<event>[^,]*)$`)
	bmcInfoFirmwareRevisionRegex        = regexp.MustCompile(`^Firmware Revision\s*:\s*(?P<value>[0-9.]*).*`)
	bmcInfoSystemFirmwareVersionRegex   = regexp.MustCompile(`^System Firmware Version\s*:\s*(?P<value>[0-9.]*).*`)
	bmcInfoManufacturerIDRegex          = regexp.MustCompile(`^Manufacturer ID\s*:\s*(?P<value>.*)`)
	bmcInfoBmcURLRegex                  = regexp.MustCompile(`^BMC URL\s*:\s*(?P<value>.*)`)
	bmcWatchdogTimerStateRegex          = regexp.MustCompile(`^Timer:\s*(?P<value>Running|Stopped)`)
	bmcWatchdogTimerUseRegex            = regexp.MustCompile(`^Timer Use:\s*(?P<value>.*)`)
	bmcWatchdogTimerLoggingRegex        = regexp.MustCompile(`^Logging:\s*(?P<value>Enabled|Disabled)`)
	bmcWatchdogTimeoutActionRegex       = regexp.MustCompile(`^Timeout Action:\s*(?P<value>.*)`)
	bmcWatchdogPretimeoutInterruptRegex = regexp.MustCompile(`^Pre-Timeout Interrupt:\s*(?P<value>.*)`)
	bmcWatchdogPretimeoutIntervalRegex  = regexp.MustCompile(`^Pre-Timeout Interval:\s*(?P<value>[0-9.]*)\s*seconds.*`)
	bmcWatchdogInitialCountdownRegex    = regexp.MustCompile(`^Initial Countdown:\s*(?P<value>[0-9.]*)\s*seconds.*`)
	bmcWatchdogCurrentCountdownRegex    = regexp.MustCompile(`^Current Countdown:\s*(?P<value>[0-9.]*)\s*seconds.*`)
)

// Result represents the outcome of a call to one of the FreeIPMI tools.
// It can be used with other functions in this package to extract data.
type Result struct {
	output []byte
	err    error
}

// SensorData represents the reading of a single sensor.
type SensorData struct {
	ID    int64
	Name  string
	Type  string
	State string
	Value float64
	Unit  string
	Event string
}

// SELEvent represents log line from SEL
type SELEventData struct {
	ID    int64
	Date  string
	Time  string
	Name  string
	Type  string
	State string
	Event string
}

// EscapePassword escapes a password so that the result is suitable for usage in a
// FreeIPMI config file.
func EscapePassword(password string) string {
	return strings.ReplaceAll(password, "#", "\\#")
}

func contains(s []int64, elm int64) bool {
	return slices.Contains(s, elm)
}

func getValue(ipmiOutput []byte, regex *regexp.Regexp) (string, error) {
	for line := range strings.SplitSeq(string(ipmiOutput), "\n") {
		match := regex.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		for i, name := range regex.SubexpNames() {
			if name != "value" {
				continue
			}
			return match[i], nil
		}
	}
	return "", fmt.Errorf("could not find value in output: %s", string(ipmiOutput))
}

// Execute runs a FreeIPMI tool with no deadline. It is a convenience wrapper
// around ExecuteContext, kept for backwards compatibility.
func Execute(cmd string, args []string, config string, target string, logger *slog.Logger) Result {
	return ExecuteContext(context.Background(), cmd, args, config, target, logger)
}

// ExecuteContext runs a FreeIPMI tool, bounded by ctx. When ctx expires, the
// tool's process group receives SIGTERM; if it still has not let go of its
// output pipes after waitDelay, it is killed.
func ExecuteContext(ctx context.Context, cmd string, args []string, config string, target string, logger *slog.Logger) Result {
	// The FreeIPMI config is handed over on an inherited anonymous pipe: the
	// read end becomes fd 3 in the child (first ExtraFiles entry), addressed
	// as /dev/fd/3 — available on every unix platform this package builds for,
	// and FreeIPMI's config parser only needs access(R_OK) + open(2). Keeps
	// credentials off the command line and the filesystem. The config is
	// passed even when empty: an empty override deliberately masks any
	// system-wide /etc/freeipmi/freeipmi.conf; omitting it would change
	// behavior.
	r, w, err := os.Pipe()
	if err != nil {
		return Result{nil, err}
	}
	defer r.Close()

	args = append(args, "--config-file", "/dev/fd/3")
	if target != "" {
		args = append(args, "-h", target)
	}

	c := exec.CommandContext(ctx, cmd, args...)
	c.ExtraFiles = []*os.File{r}
	// Run the tool in its own process group so cancellation can signal the
	// whole group, not just the direct child.
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		err := syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	// Also bounds how long Wait blocks on output pipes held open by an
	// orphaned grandchild after the child itself has exited.
	c.WaitDelay = waitDelay

	// Feed the config from a goroutine rather than up front: its size is
	// unbounded (usernames, passwords, workaround flags), so writing before
	// start could deadlock on the pipe buffer. The goroutine cannot leak —
	// once both read ends are gone (child exited, deferred r.Close above), a
	// blocked write fails with EPIPE and the goroutine exits.
	go func() {
		if _, err := w.Write([]byte(config)); err != nil && !errors.Is(err, syscall.EPIPE) {
			logger.Error("Error writing config to pipe", "error", err)
		}
		w.Close()
	}()

	logger.Debug("Executing", "command", cmd, "args", fmt.Sprintf("%+v", args))
	start := time.Now()
	out, err := c.CombinedOutput()
	if err != nil {
		// Distinguish a deadline kill from an ordinary non-zero exit so
		// operators can tell a slow BMC from a failing tool.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("error running %s: timed out after %s", cmd, time.Since(start).Round(time.Millisecond))
		} else {
			err = fmt.Errorf("error running %s: %s", cmd, err)
		}
	}
	return Result{out, err}
}

func GetSensorData(ipmiOutput Result, excludeSensorIDs []int64) ([]SensorData, error) {
	var result []SensorData

	if ipmiOutput.err != nil {
		return result, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}

	r := csv.NewReader(bytes.NewReader(ipmiOutput.output))
	fields, err := r.ReadAll()
	if err != nil {
		return result, err
	}

	for _, line := range fields {
		var data SensorData

		data.ID, err = strconv.ParseInt(line[0], 10, 64)
		if err != nil {
			return result, err
		}
		if contains(excludeSensorIDs, data.ID) {
			continue
		}

		data.Name = line[1]
		data.Type = line[2]
		data.State = line[3]

		value := line[4]
		if value != "N/A" {
			data.Value, err = strconv.ParseFloat(value, 64)
			if err != nil {
				return result, err
			}
		} else {
			data.Value = math.NaN()
		}

		data.Unit = line[5]
		data.Event = strings.Trim(line[6], "'")

		result = append(result, data)
	}
	return result, err
}

func GetCurrentPowerConsumption(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	// Check for Power Measurement are avail
	value, err := getValue(ipmiOutput.output, ipmiDCMIPowerMeasurementRegex)
	if err != nil {
		return -1, err
	}
	// When Power Measurement in 'Active' state - we can get watts
	if value == "Active" {
		value, err := getValue(ipmiOutput.output, ipmiDCMICurrentPowerRegex)
		if err != nil {
			return -1, err
		}
		return strconv.ParseFloat(value, 64)
	}
	return -1, nil
}

func GetChassisPowerState(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, ipmiChassisPowerRegex)
	if err != nil {
		return -1, err
	}
	if value == "on" {
		return 1, err
	}
	return 0, err
}

func GetChassisDriveFault(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, ipmiChassisDriveFaultRegex)
	if err != nil {
		return -1, err
	}
	if value == "false" {
		return 1, err
	}
	return 0, err
}

func GetChassisCoolingFault(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, ipmiChassisCoolingFaultRegex)
	if err != nil {
		return -1, err
	}
	if value == "false" {
		return 1, err
	}
	return 0, err
}

func GetBMCInfoFirmwareRevision(ipmiOutput Result) (string, error) {
	// Workaround for an issue described here: https://github.com/prometheus-community/ipmi_exporter/issues/57
	// The command may fail, but produce usable output (minus the system firmware revision).
	// Try to recover gracefully from that situation by first trying to parse the output, and only
	// raise the initial error if that also fails.
	value, err := getValue(ipmiOutput.output, bmcInfoFirmwareRevisionRegex)
	if err != nil {
		if ipmiOutput.err != nil {
			return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
		}
	}
	return value, err
}

func GetBMCInfoManufacturerID(ipmiOutput Result) (string, error) {
	// Workaround for an issue described here: https://github.com/prometheus-community/ipmi_exporter/issues/57
	// The command may fail, but produce usable output (minus the system firmware revision).
	// Try to recover gracefully from that situation by first trying to parse the output, and only
	// raise the initial error if that also fails.
	value, err := getValue(ipmiOutput.output, bmcInfoManufacturerIDRegex)
	if err != nil {
		if ipmiOutput.err != nil {
			return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
		}
	}
	return value, err
}

func GetBMCInfoSystemFirmwareVersion(ipmiOutput Result) (string, error) {
	if ipmiOutput.err != nil {
		return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	return getValue(ipmiOutput.output, bmcInfoSystemFirmwareVersionRegex)
}

func GetBMCInfoBmcURL(ipmiOutput Result) (string, error) {
	if ipmiOutput.err != nil {
		return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	return getValue(ipmiOutput.output, bmcInfoBmcURLRegex)
}

func GetSELInfoEntriesCount(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, ipmiSELEntriesRegex)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(value, 64)
}

func GetSELInfoFreeSpace(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, ipmiSELFreeSpaceRegex)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(value, 64)
}

func GetRawOctets(ipmiOutput Result) ([]string, error) {
	if ipmiOutput.err != nil {
		return nil, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	strOutput := strings.Trim(string(ipmiOutput.output), " \r\n")
	if !strings.HasPrefix(strOutput, "rcvd: ") {
		return nil, fmt.Errorf("unexpected raw response: %s", strOutput)
	}
	octets := strings.Split(strOutput[6:], " ")
	return octets, nil
}

func GetBMCWatchdogTimerState(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, bmcWatchdogTimerStateRegex)
	if err != nil {
		return -1, err
	}
	if value == "Running" {
		return 1, err
	}
	return 0, err
}

func GetBMCWatchdogTimerUse(ipmiOutput Result) (string, error) {
	if ipmiOutput.err != nil {
		return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	return getValue(ipmiOutput.output, bmcWatchdogTimerUseRegex)
}

func GetBMCWatchdogLoggingState(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, bmcWatchdogTimerLoggingRegex)
	if err != nil {
		return -1, err
	}
	if value == "Enabled" {
		return 1, err
	}
	return 0, err
}

func GetBMCWatchdogTimeoutAction(ipmiOutput Result) (string, error) {
	if ipmiOutput.err != nil {
		return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	return getValue(ipmiOutput.output, bmcWatchdogTimeoutActionRegex)
}

func GetBMCWatchdogPretimeoutInterrupt(ipmiOutput Result) (string, error) {
	if ipmiOutput.err != nil {
		return "", fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	return getValue(ipmiOutput.output, bmcWatchdogPretimeoutInterruptRegex)
}

func GetBMCWatchdogPretimeoutInterval(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, bmcWatchdogPretimeoutIntervalRegex)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(value, 64)
}

func GetBMCWatchdogInitialCountdown(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, bmcWatchdogInitialCountdownRegex)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(value, 64)
}

func GetBMCWatchdogCurrentCountdown(ipmiOutput Result) (float64, error) {
	if ipmiOutput.err != nil {
		return -1, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}
	value, err := getValue(ipmiOutput.output, bmcWatchdogCurrentCountdownRegex)
	if err != nil {
		return -1, err
	}
	return strconv.ParseFloat(value, 64)
}

func GetSELEvents(ipmiOutput Result) ([]SELEventData, error) {
	if ipmiOutput.err != nil {
		return nil, fmt.Errorf("%s: %s", ipmiOutput.err, ipmiOutput.output)
	}

	scanner := bufio.NewScanner(bytes.NewReader(ipmiOutput.output))
	events := []SELEventData{}
	for scanner.Scan() {
		line := scanner.Text()
		match := ipmiSELEventRegex.FindStringSubmatch(line)
		// ignore lines which does not matches event regexp
		if match == nil {
			continue
		}

		result := make(map[string]string)
		for i, name := range ipmiSELEventRegex.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}
		id, err := strconv.ParseInt(result["id"], 10, 64)

		// ignore lines which does not starts with number
		if err != nil {
			continue
		}

		events = append(events, SELEventData{
			ID:    id,
			Date:  result["date"],
			Time:  result["time"],
			Name:  result["name"],
			Type:  result["type"],
			State: result["state"],
			Event: result["event"],
		})
	}
	return events, nil
}
