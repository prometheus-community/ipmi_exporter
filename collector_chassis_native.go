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
	chassisNativePowerStateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_state"),
		"Current power state (1=on, 0=off).",
		[]string{},
		nil,
	)
	chassisNativeDriveFaultDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "drive_fault_state"),
		"Current drive fault state (1=true, 0=false).", // TODO value mapping changed
		[]string{},
		nil,
	)
	chassisNativeCoolingFaultDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "cooling_fault_state"),
		"Current Cooling/fan fault state (1=true, 0=false).", // TODO value mapping changed
		[]string{},
		nil,
	)
)

type ChassisNativeCollector struct{}

func (c ChassisNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return ChassisCollectorName
}

func (c ChassisNativeCollector) Cmd() string {
	return ""
}

func (c ChassisNativeCollector) Args() []string {
	return []string{}
}

func (c ChassisNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	client, err := NewNativeClient(target)
	if err != nil {
		return 0, err
	}
	res, err := client.GetChassisStatus(context.TODO())
	if err != nil {
		return 0, err
	}

	ch <- prometheus.MustNewConstMetric(
		chassisNativePowerStateDesc,
		prometheus.GaugeValue,
		boolToFloat(res.PowerIsOn),
	)
	ch <- prometheus.MustNewConstMetric(
		chassisNativeDriveFaultDesc,
		prometheus.GaugeValue,
		boolToFloat(res.DriveFault),
	)
	ch <- prometheus.MustNewConstMetric(
		chassisNativeCoolingFaultDesc,
		prometheus.GaugeValue,
		boolToFloat(res.CollingFanFault),
	)
	return 1, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
