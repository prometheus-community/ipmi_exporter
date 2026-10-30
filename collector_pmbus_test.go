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
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
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
	"go.yaml.in/yaml/v2"
)

// pmbusTestAnswers are a real power supply's answers, read through a shelf's
// BMC.
var pmbusTestAnswers = map[uint8][]byte{
	pmbusVoutMode:  {0x17},
	pmbusReadVin:   {0xc7, 0xf1},
	pmbusReadIin:   {0x59, 0xc1},
	pmbusReadVout:  {0x8c, 0x18},
	pmbusReadIout:  {0xa1, 0xd2},
	pmbusReadTemp1: {0x2a, 0x00},
	pmbusReadTemp2: {0x30, 0x00},
	pmbusReadFan1:  {0xe4, 0x28},
	pmbusReadPout:  {0x08, 0xf2},
	pmbusReadPin:   {0x64, 0xf2},
	pmbusMfrID:     pmbusTestText("\x0aSUPERMICRO\xd9"),
	pmbusMfrModel:  pmbusTestText("\x0cPWS-1K23A-1R0"),
	pmbusMfrSerial: pmbusTestText("\x0fP1K2ACN52PB2493\x9c"),
}

// pmbusTestText pads an answer to the 32 bytes asked for, as the BMC does.
func pmbusTestText(s string) []byte {
	return append([]byte(s), bytes.Repeat([]byte{0xff}, 32-len(s))...)
}

// pmbusTestSupplies are one power supply that answers and one that doesn't.
var pmbusTestSupplies = []PowerSupplyConfig{{Bus: 0x03, Address: 0x78}, {Name: "rear", Bus: 0x03, Address: 0x7a}}

// pmbusExpected is what both versions report for them.
const pmbusExpected = `
# HELP ipmi_psu_fan_speed_rpm Speed of a power supply's fan in rotations per minute.
# TYPE ipmi_psu_fan_speed_rpm gauge
ipmi_psu_fan_speed_rpm{psu="1"} 7296
# HELP ipmi_psu_info Constant metric with value '1' naming a power supply's maker, model and serial number.
# TYPE ipmi_psu_info gauge
ipmi_psu_info{manufacturer="SUPERMICRO",model="PWS-1K23A-1R",psu="1",serial="P1K2ACN52PB2493"} 1
# HELP ipmi_psu_input_amperes Current a power supply draws at its input, in amperes.
# TYPE ipmi_psu_input_amperes gauge
ipmi_psu_input_amperes{psu="1"} 1.34765625
# HELP ipmi_psu_input_volts Voltage at a power supply's input.
# TYPE ipmi_psu_input_volts gauge
ipmi_psu_input_volts{psu="1"} 113.75
# HELP ipmi_psu_input_watts Power a power supply draws at its input, in watts.
# TYPE ipmi_psu_input_watts gauge
ipmi_psu_input_watts{psu="1"} 153
# HELP ipmi_psu_output_amperes Current a power supply delivers on its main output, in amperes.
# TYPE ipmi_psu_output_amperes gauge
ipmi_psu_output_amperes{psu="1"} 10.515625
# HELP ipmi_psu_output_volts Voltage of a power supply's main output.
# TYPE ipmi_psu_output_volts gauge
ipmi_psu_output_volts{psu="1"} 12.2734375
# HELP ipmi_psu_output_watts Power a power supply delivers at its output, in watts.
# TYPE ipmi_psu_output_watts gauge
ipmi_psu_output_watts{psu="1"} 130
# HELP ipmi_psu_temperature_celsius Temperature inside a power supply, in degrees Celsius.
# TYPE ipmi_psu_temperature_celsius gauge
ipmi_psu_temperature_celsius{psu="1",sensor="1"} 42
ipmi_psu_temperature_celsius{psu="1",sensor="2"} 48
# HELP ipmi_up '1' if a scrape of the IPMI device was successful, '0' otherwise.
# TYPE ipmi_up gauge
ipmi_up{collector="pmbus"} 1
`

var pmbusMetricNames = []string{
	"ipmi_psu_fan_speed_rpm", "ipmi_psu_info", "ipmi_psu_input_amperes", "ipmi_psu_input_volts", "ipmi_psu_input_watts",
	"ipmi_psu_output_amperes", "ipmi_psu_output_volts", "ipmi_psu_output_watts", "ipmi_psu_temperature_celsius", "ipmi_up",
}

// pmbusTestCollector is the exporter's collector for a module running pmbus.
func pmbusTestCollector(t *testing.T, target string, config IPMIConfig, native bool) metaCollector {
	t.Helper()

	oldLogger, oldNative := logger, *nativeIPMI
	t.Cleanup(func() { logger, *nativeIPMI = oldLogger, oldNative })
	logger, *nativeIPMI = slog.New(slog.DiscardHandler), native

	config.Collectors = []CollectorName{PMBusCollectorName}
	config.PowerSupplies = pmbusTestSupplies
	config.PowerSupplyInfo = true
	return metaCollector{target: target, module: "test", config: &SafeConfig{C: &Config{Modules: map[string]IPMIConfig{"test": config}}}}
}

func TestPMBusCollector(t *testing.T) {
	// A stand-in for ipmi-raw. It needs the custom args.
	var cases strings.Builder
	for command, answer := range pmbusTestAnswers {
		fmt.Fprintf(&cases, "\t0x%02x) echo \"rcvd: 52 00 % X \" ;;\n", command, answer)
	}
	script := `#!/bin/sh
[ "$1" = "--custom" ] || exit 1
while read -r lun netfn cmd bus address length command; do
	if [ "$lun $netfn $cmd $bus $address" != "0x00 0x06 0x52 0x03 0x78" ]; then
		echo "rcvd: 52 83 "
		continue
	fi
	case "$command" in
` + cases.String() + `	*) echo "rcvd: 52 C1 " ;;
	esac
done
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ipmi-raw"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := *executablesPath
	t.Cleanup(func() { *executablesPath = oldPath })
	*executablesPath = dir

	c := pmbusTestCollector(t, targetLocal, IPMIConfig{CustomArgs: map[CollectorName][]string{PMBusCollectorName: {"--custom"}}}, false)
	if err := testutil.CollectAndCompare(c, strings.NewReader(pmbusExpected), pmbusMetricNames...); err != nil {
		t.Error(err)
	}
}

func TestPMBusNativeCollector(t *testing.T) {
	// A simulated BMC with the same power supplies.
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
	reg.RegisterFunc(ipmitypes.Command{ID: 0x52, NetFn: ipmitypes.NetFnAppRequest}, func(_ context.Context, _ *handlers.HandlerContext, data []byte) ([]byte, ipmitypes.CompletionCode, error) {
		if len(data) != 4 || data[0] != 0x03 || data[1] != 0x78 {
			return nil, ipmitypes.CompletionCode(0x83), nil
		}
		answer, ok := pmbusTestAnswers[data[3]]
		if !ok || len(answer) != int(data[2]) {
			return nil, ipmitypes.CodeInvalidCommand, nil
		}
		return answer, ipmitypes.CodeOK, nil
	})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = server.NewServer(b, udp.Wrap(conn), server.WithHandlerRegistry(reg)).Serve(ctx) }()

	c := pmbusTestCollector(t, conn.LocalAddr().String(), IPMIConfig{User: "admin", Password: "secret"}, true)
	if err := testutil.CollectAndCompare(c, strings.NewReader(pmbusExpected), pmbusMetricNames...); err != nil {
		t.Error(err)
	}
}

func TestPowerSuppliesConfig(t *testing.T) {
	var c Config
	config := "modules:\n  shelf:\n    collectors: [pmbus]\n    power_supplies:\n    - bus: 0x03\n      address: 0x78\n    - name: rear\n      bus: 0x03\n      address: 0x7a\n"
	if err := yaml.Unmarshal([]byte(config), &c); err != nil {
		t.Fatal(err)
	}

	want := []PowerSupplyConfig{{Bus: 0x03, Address: 0x78}, {Name: "rear", Bus: 0x03, Address: 0x7a}}
	if got := c.Modules["shelf"].PowerSupplies; !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
