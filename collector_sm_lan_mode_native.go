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

package main

import (
	"context"
	"fmt"

	"github.com/bougou/go-ipmi"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

var (
	lanModeNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "config", "lan_mode"),
		"Returns configured LAN mode (0=dedicated, 1=shared, 2=failover).",
		nil,
		nil,
	)
)

type SMLANModeNativeCollector struct{}

func (c SMLANModeNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return SMLANModeCollectorName
}

func (c SMLANModeNativeCollector) Cmd() string {
	return ""
}

func (c SMLANModeNativeCollector) Args() []string {
	return []string{}
}

func (c SMLANModeNativeCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	client, err := NewNativeClient(target)
	if err != nil {
		return 0, err
	}
	if _, err := client.SetSessionPrivilegeLevel(context.TODO(), ipmi.PrivilegeLevelAdministrator); err != nil {
		logger.Error("Failed to set privilege level to admin", "target", targetName(target.host))
		return 0, fmt.Errorf("failed to set privilege level to admin")
	}
	res, err := client.RawCommand(context.TODO(), ipmi.NetFnOEMSupermicroRequest, 0x70, []byte{0x0C, 0x00}, "GetSupermicroLanMode")
	if err != nil {
		logger.Error("raw command failed", "error", err)
		return 0, err
	}

	if len(res.Response) != 1 {
		logger.Error("Unexpected number of octets", "target", targetName(target.host), "octets", len(res.Response))
		return 0, fmt.Errorf("unexpected number of octets in raw response: %d", len(res.Response))
	}

	value := res.Response[0]
	switch value {
	case 0, 1, 2:
		ch <- prometheus.MustNewConstMetric(lanModeNativeDesc, prometheus.GaugeValue, float64(value))
	default:
		logger.Error("Unexpected lan mode status (ipmi-raw)", "target", targetName(target.host), "status", value)
		return 0, fmt.Errorf("unexpected lan mode status: %d", value)
	}

	return 1, nil
}
