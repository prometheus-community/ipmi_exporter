// Copyright 2025 The Prometheus Authors
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
	"strings"
	"testing"

	"github.com/bougou/go-ipmi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// watchdogTimerCollector adapts collectWatchdogTimerNative to the
// prometheus.Collector interface, so that a decoded Get Watchdog Timer response
// can be turned into metrics without talking to a BMC.
type watchdogTimerCollector struct {
	res *ipmi.GetWatchdogTimerResponse
}

func (c watchdogTimerCollector) Describe(_ chan<- *prometheus.Desc) {}

func (c watchdogTimerCollector) Collect(ch chan<- prometheus.Metric) {
	collectWatchdogTimerNative(ch, c.res)
}

// TestCollectWatchdogTimerNativeCountdownUnits checks the unit conversion of the
// countdown values in the Get Watchdog Timer response (IPMI v2.0 section 27.7):
// the pre-timeout interval is in seconds, but the initial and present countdown
// values are counts of 100ms units and must be divided by 10 before being
// exposed on the _seconds metrics.
func TestCollectWatchdogTimerNativeCountdownUnits(t *testing.T) {
	cases := []struct {
		name string
		// raw is the data part of a Get Watchdog Timer response, i.e. what the
		// BMC returns after the completion code.
		raw  []byte
		want string
	}{
		{
			// A 60s timer with 51.5s left, as reported by a Supermicro X9
			// (initial countdown 600, present countdown 515, both in 100ms
			// units), plus a 30s pre-timeout interval, which is in seconds.
			name: "running timer",
			raw:  []byte{0x44, 0x01, 0x1e, 0x00, 0x58, 0x02, 0x03, 0x02},
			want: `
# HELP ipmi_bmc_watchdog_current_countdown_seconds Watchdog current countdown in seconds
# TYPE ipmi_bmc_watchdog_current_countdown_seconds gauge
ipmi_bmc_watchdog_current_countdown_seconds 51.5
# HELP ipmi_bmc_watchdog_initial_countdown_seconds Watchdog initial countdown in seconds
# TYPE ipmi_bmc_watchdog_initial_countdown_seconds gauge
ipmi_bmc_watchdog_initial_countdown_seconds 60
# HELP ipmi_bmc_watchdog_pretimeout_interval_seconds Watchdog pre-timeout interval in seconds
# TYPE ipmi_bmc_watchdog_pretimeout_interval_seconds gauge
ipmi_bmc_watchdog_pretimeout_interval_seconds 30
`,
		},
		{
			// Expired timer: present countdown 0, initial countdown 1200
			// (= 120s), no pre-timeout interval configured.
			name: "expired timer",
			raw:  []byte{0x04, 0x01, 0x00, 0x02, 0xb0, 0x04, 0x00, 0x00},
			want: `
# HELP ipmi_bmc_watchdog_current_countdown_seconds Watchdog current countdown in seconds
# TYPE ipmi_bmc_watchdog_current_countdown_seconds gauge
ipmi_bmc_watchdog_current_countdown_seconds 0
# HELP ipmi_bmc_watchdog_initial_countdown_seconds Watchdog initial countdown in seconds
# TYPE ipmi_bmc_watchdog_initial_countdown_seconds gauge
ipmi_bmc_watchdog_initial_countdown_seconds 120
# HELP ipmi_bmc_watchdog_pretimeout_interval_seconds Watchdog pre-timeout interval in seconds
# TYPE ipmi_bmc_watchdog_pretimeout_interval_seconds gauge
ipmi_bmc_watchdog_pretimeout_interval_seconds 0
`,
		},
		{
			// Maximum countdown value: 65535 * 100ms.
			name: "maximum countdown",
			raw:  []byte{0x44, 0x01, 0xff, 0x00, 0xff, 0xff, 0xff, 0xff},
			want: `
# HELP ipmi_bmc_watchdog_current_countdown_seconds Watchdog current countdown in seconds
# TYPE ipmi_bmc_watchdog_current_countdown_seconds gauge
ipmi_bmc_watchdog_current_countdown_seconds 6553.5
# HELP ipmi_bmc_watchdog_initial_countdown_seconds Watchdog initial countdown in seconds
# TYPE ipmi_bmc_watchdog_initial_countdown_seconds gauge
ipmi_bmc_watchdog_initial_countdown_seconds 6553.5
# HELP ipmi_bmc_watchdog_pretimeout_interval_seconds Watchdog pre-timeout interval in seconds
# TYPE ipmi_bmc_watchdog_pretimeout_interval_seconds gauge
ipmi_bmc_watchdog_pretimeout_interval_seconds 255
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := &ipmi.GetWatchdogTimerResponse{}
			if err := res.Unpack(tc.raw); err != nil {
				t.Fatalf("failed to unpack response: %v", err)
			}
			if err := testutil.CollectAndCompare(
				watchdogTimerCollector{res: res},
				strings.NewReader(tc.want),
				"ipmi_bmc_watchdog_pretimeout_interval_seconds",
				"ipmi_bmc_watchdog_initial_countdown_seconds",
				"ipmi_bmc_watchdog_current_countdown_seconds",
			); err != nil {
				t.Error(err)
			}
		})
	}
}
