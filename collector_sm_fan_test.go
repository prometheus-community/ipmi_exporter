// Copyright The Prometheus Authors
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

package main

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bougou/go-ipmi/pkg/bmc"
	"github.com/bougou/go-ipmi/pkg/clock"
	"github.com/bougou/go-ipmi/pkg/hal/mock"
	"github.com/bougou/go-ipmi/pkg/handlers"
	"github.com/bougou/go-ipmi/pkg/server"
	"github.com/bougou/go-ipmi/pkg/transport/udp"
	ipmitypes "github.com/bougou/go-ipmi/pkg/types"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// smFanExpected is what both versions report for a BMC in full mode, with zone
// 0 at 50% and no zone 1.
const smFanExpected = `
# HELP ipmi_config_fan_mode Returns configured fan mode (0=standard, 1=full, 2=optimal, 3=PUE, 4=heavy IO).
# TYPE ipmi_config_fan_mode gauge
ipmi_config_fan_mode 1
# HELP ipmi_fan_zone_duty_ratio Fan duty of a fan zone as a proportion of full duty.
# TYPE ipmi_fan_zone_duty_ratio gauge
ipmi_fan_zone_duty_ratio{zone="0"} 0.5
# HELP ipmi_up '1' if a scrape of the IPMI device was successful, '0' otherwise.
# TYPE ipmi_up gauge
ipmi_up{collector="sm-fan"} 1
`

// smFanTestCollector is the exporter's collector for a module running sm-fan.
func smFanTestCollector(t *testing.T, target string, config IPMIConfig, native bool) metaCollector {
	t.Helper()

	oldLogger, oldNative := logger, *nativeIPMI
	t.Cleanup(func() { logger, *nativeIPMI = oldLogger, oldNative })
	logger, *nativeIPMI = slog.New(slog.DiscardHandler), native

	config.Collectors = []CollectorName{SMFanCollectorName}
	return metaCollector{target: target, module: "test", config: &SafeConfig{C: &Config{Modules: map[string]IPMIConfig{"test": config}}}}
}

func TestSMFanCollector(t *testing.T) {
	// A stand-in for ipmi-raw. It needs the custom args.
	dir := t.TempDir()
	script := `#!/bin/sh
[ "$1" = "--custom" ] || exit 1
while read -r line; do
	case "$line" in
	"0x00 0x30 0x45 0x00") echo "rcvd: 45 00 01 " ;;
	"0x00 0x30 0x70 0x66 0x00 0x00") echo "rcvd: 70 00 32 " ;;
	*) echo "rcvd: 70 C1 " ;;
	esac
done
`
	if err := os.WriteFile(filepath.Join(dir, "ipmi-raw"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := *executablesPath
	t.Cleanup(func() { *executablesPath = oldPath })
	*executablesPath = dir

	c := smFanTestCollector(t, targetLocal, IPMIConfig{CustomArgs: map[CollectorName][]string{SMFanCollectorName: {"--custom"}}}, false)
	if err := testutil.CollectAndCompare(c, strings.NewReader(smFanExpected), "ipmi_config_fan_mode", "ipmi_fan_zone_duty_ratio", "ipmi_up"); err != nil {
		t.Error(err)
	}
}

func TestSMFanNativeCollector(t *testing.T) {
	// A simulated BMC with no zone 1.
	b := bmc.New(bmc.DeviceInfo{DeviceID: 1, IPMIVersion: 0x20}, [16]byte{}, mock.New(), bmc.WithClock(clock.Real))
	user, err := b.Users.Add(2, "admin")
	if err != nil {
		t.Fatal(err)
	}
	user.SetPassword([]byte("secret"))
	user.Enabled = true
	user.ChannelAccess[1] = bmc.UserChannelAccess{MaxPrivilege: bmc.PrivilegeLevelAdministrator, Enabled: true}

	reg := handlers.NewRegistry()
	handlers.RegisterAllHandlers(reg)
	reg.RegisterFunc(ipmitypes.Command{ID: 0x45, NetFn: ipmitypes.NetFnOEMSupermicroRequest}, func(context.Context, *handlers.HandlerContext, []byte) ([]byte, ipmitypes.CompletionCode, error) {
		return []byte{0x01}, ipmitypes.CodeOK, nil
	})
	reg.RegisterFunc(ipmitypes.Command{ID: 0x70, NetFn: ipmitypes.NetFnOEMSupermicroRequest}, func(_ context.Context, _ *handlers.HandlerContext, data []byte) ([]byte, ipmitypes.CompletionCode, error) {
		if bytes.Equal(data, []byte{0x66, 0x00, 0x00}) {
			return []byte{0x32}, ipmitypes.CodeOK, nil
		}
		return nil, ipmitypes.CodeInvalidCommand, nil
	})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = server.NewServer(b, udp.Wrap(conn), server.WithHandlerRegistry(reg)).Serve(ctx) }()

	c := smFanTestCollector(t, conn.LocalAddr().String(), IPMIConfig{User: "admin", Password: "secret"}, true)
	if err := testutil.CollectAndCompare(c, strings.NewReader(smFanExpected), "ipmi_config_fan_mode", "ipmi_fan_zone_duty_ratio", "ipmi_up"); err != nil {
		t.Error(err)
	}
}
