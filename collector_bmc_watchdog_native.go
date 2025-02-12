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
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

var (
	bmcWatchdogNativeTimerDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "timer_state"),
		"Watchdog timer running (1: running, 0: stopped)",
		[]string{},
		nil,
	)
	// TODO add "Reserved" (0x0)? also needed in lib
	watchdogNativeTimerUses       = []string{"BIOS FRB2", "BIOS/POST", "OS Load", "SMS/OS", "OEM"}
	bmcWatchdogNativeTimerUseDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "timer_use_state"),
		"Watchdog timer use (1: active, 0: inactive)",
		[]string{"name"},
		nil,
	)
	bmcWatchdogNativeLoggingDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "logging_state"),
		"Watchdog log flag (1: Enabled, 0: Disabled / note: reverse of freeipmi)",
		[]string{},
		nil,
	)
	watchdogNativeTimeoutActions       = []string{"No action", "Hard Reset", "Power Down", "Power Cycle"}
	bmcWatchdogNativeTimeoutActionDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "timeout_action_state"),
		"Watchdog timeout action (1: active, 0: inactive)",
		[]string{"action"},
		nil,
	)
	watchdogNativePretimeoutInterrupts       = []string{"None", "SMI", "NMI / Diagnostic Interrupt", "Messaging Interrupt"}
	bmcWatchdogNativePretimeoutInterruptDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "pretimeout_interrupt_state"),
		"Watchdog pre-timeout interrupt (1: active, 0: inactive)",
		[]string{"interrupt"},
		nil,
	)
	bmcWatchdogNativePretimeoutIntervalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "pretimeout_interval_seconds"),
		"Watchdog pre-timeout interval in seconds",
		[]string{},
		nil,
	)
	bmcWatchdogNativeInitialCountdownDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "initial_countdown_seconds"),
		"Watchdog initial countdown in seconds",
		[]string{},
		nil,
	)
	bmcWatchdogNativeCurrentCountdownDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc_watchdog", "current_countdown_seconds"),
		"Watchdog initial countdown in seconds",
		[]string{},
		nil,
	)
)

type BMCWatchdogNativeCollector struct{}

func (c BMCWatchdogNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return BMCWatchdogCollectorName
}

func (c BMCWatchdogNativeCollector) Cmd() string {
	return ""
}

func (c BMCWatchdogNativeCollector) Args() []string {
	return []string{}
}

func (c BMCWatchdogNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {

	// TODO this now works remotely

	client, err := NewNativeClient(target)
	if err != nil {
		return 0, err
	}
	res, err := client.GetWatchdogTimer(context.TODO())
	if err != nil {
		return 0, err
	}

	ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeTimerDesc, prometheus.GaugeValue, boolToFloat(res.TimerIsStarted))
	for _, timerUse := range watchdogNativeTimerUses {
		if res.TimerUse.String() == timerUse {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeTimerUseDesc, prometheus.GaugeValue, 1, timerUse)
		} else {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeTimerUseDesc, prometheus.GaugeValue, 0, timerUse)
		}
	}
	ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeLoggingDesc, prometheus.GaugeValue, boolToFloat(!res.DontLog))
	for _, timeoutAction := range watchdogNativeTimeoutActions {
		if res.TimeoutAction.String() == timeoutAction {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeTimeoutActionDesc, prometheus.GaugeValue, 1, timeoutAction)
		} else {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeTimeoutActionDesc, prometheus.GaugeValue, 0, timeoutAction)
		}
	}
	for _, pretimeoutInterrupt := range watchdogNativePretimeoutInterrupts {
		if res.PreTimeoutInterrupt.String() == pretimeoutInterrupt {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativePretimeoutInterruptDesc, prometheus.GaugeValue, 1, pretimeoutInterrupt)
		} else {
			ch <- prometheus.MustNewConstMetric(bmcWatchdogNativePretimeoutInterruptDesc, prometheus.GaugeValue, 0, pretimeoutInterrupt)
		}
	}
	ch <- prometheus.MustNewConstMetric(bmcWatchdogNativePretimeoutIntervalDesc, prometheus.GaugeValue, float64(res.PreTimeoutIntervalSec))
	ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeInitialCountdownDesc, prometheus.GaugeValue, float64(res.InitialCountdown))
	ch <- prometheus.MustNewConstMetric(bmcWatchdogNativeCurrentCountdownDesc, prometheus.GaugeValue, float64(res.PresentCountdown))
	return 1, nil
}
