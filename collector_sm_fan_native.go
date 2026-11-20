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
	"context"
	"fmt"
	"strconv"

	ipmitypes "github.com/bougou/go-ipmi/pkg/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

type SMFanNativeCollector struct{}

func (c SMFanNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return SMFanCollectorName
}

func (c SMFanNativeCollector) Cmd() string {
	return ""
}

func (c SMFanNativeCollector) Args() []string {
	return []string{}
}

func (c SMFanNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)
	if _, err := client.SetSessionPrivilegeLevel(ctx, ipmitypes.PrivilegeLevelAdministrator); err != nil {
		logger.Error("Failed to set privilege level to admin", "target", targetName(target.host))
		return 0, fmt.Errorf("failed to set privilege level to admin")
	}

	res, err := client.RawCommand(ctx, ipmitypes.NetFnOEMSupermicroRequest, 0x45, []byte{0x00}, "GetSupermicroFanMode")
	if err != nil {
		logger.Error("Failed to collect fan mode data", "target", targetName(target.host), "error", err)
		return 0, err
	}
	if len(res.Response) != 1 {
		logger.Error("Unexpected number of octets", "target", targetName(target.host), "octets", len(res.Response))
		return 0, fmt.Errorf("unexpected number of octets in raw response: %d", len(res.Response))
	}
	ch <- prometheus.MustNewConstMetric(fanModeDesc, prometheus.GaugeValue, float64(res.Response[0]))

	for _, zone := range smFanZones {
		res, err := client.RawCommand(ctx, ipmitypes.NetFnOEMSupermicroRequest, 0x70, []byte{0x66, 0x00, zone}, "GetSupermicroFanZoneDuty")
		if err != nil || len(res.Response) != 1 {
			// Not every board has every zone, so skip one that doesn't answer.
			logger.Debug("Skipping fan zone", "target", targetName(target.host), "zone", zone, "error", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(fanZoneDutyDesc, prometheus.GaugeValue, float64(res.Response[0])/100, strconv.Itoa(int(zone)))
	}

	return 1, nil
}
