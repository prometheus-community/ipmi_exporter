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
	"strconv"

	"github.com/bougou/go-ipmi"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

var (
	bmcNativeInfoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc", "info"),
		"Constant metric with value '1' providing details about the BMC.",
		[]string{"firmware_revision", "manufacturer", "manufacturer_id", "system_firmware_version"},
		nil,
	)
)

type BMCNativeCollector struct{}

func (c BMCNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return BMCCollectorName
}

func (c BMCNativeCollector) Cmd() string {
	return "" // native collector => empty command
}

func (c BMCNativeCollector) Args() []string {
	return []string{}
}

func (c BMCNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	client, err := NewNativeClient(target)
	if err != nil {
		return 0, err
	}
	res, err := client.GetDeviceID(context.TODO())
	if err != nil {
		return 0, err
	}

	// The API looks slightly awkward here, but doing this instead of calling
	// client.GetSystemInfo() greatly reduces the number of required round-trips.
	systemInfo := ipmi.SystemInfoParams{
		SetInProgress: &ipmi.SystemInfoParam_SetInProgress{
			Value: ipmi.SetInProgress_SetComplete,
		},
		SystemFirmwareVersions: make([]*ipmi.SystemInfoParam_SystemFirmwareVersion, 0),
	}
	err = client.GetSystemInfoParamsFor(context.TODO(), &systemInfo)
	// This one is not always available
	systemFirmwareVersion := "N/A"
	if err != nil {
		logger.Debug("Failed to get system firmware version", "target", targetName(target.host), "error", err)
	} else {
		systemFirmwareVersion = systemInfo.ToSystemInfo().SystemFirmwareVersion
	}

	ch <- prometheus.MustNewConstMetric(
		bmcNativeInfoDesc,
		prometheus.GaugeValue,
		1,
		res.FirmwareVersionStr(),
		ipmi.OEM(res.ManufacturerID).String(),
		strconv.FormatUint(uint64(res.ManufacturerID), 10),
		systemFirmwareVersion,
	)
	return 1, nil
}
