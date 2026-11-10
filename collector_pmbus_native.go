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

	ipmitypes "github.com/bougou/go-ipmi/pkg/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

type PMBusNativeCollector struct{}

func (c PMBusNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return PMBusCollectorName
}

func (c PMBusNativeCollector) Cmd() string {
	return ""
}

func (c PMBusNativeCollector) Args() []string {
	return []string{}
}

func (c PMBusNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	supplies := target.config.PowerSupplies
	if len(supplies) == 0 {
		logger.Error("No power supplies configured", "target", targetName(target.host))
		return 0, fmt.Errorf("no power supplies configured")
	}

	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)

	answered := false
	for i, ps := range supplies {
		answers := map[uint8][]byte{}
		for _, r := range pmbusReadsFor(target.config) {
			// Ask the power supply through the BMC.
			res, err := client.RawCommand(ctx, ipmitypes.NetFnAppRequest, 0x52, []byte{ps.Bus, ps.Address, r.length, r.command}, "MasterWriteRead")
			if err != nil {
				logger.Debug("Power supply did not answer", "target", targetName(target.host), "psu", pmbusName(i, ps), "command", r.command, "error", err)
				continue
			}
			answers[r.command] = res.Response
		}
		if collectPowerSupply(ch, pmbusName(i, ps), answers) {
			answered = true
		}
	}
	if !answered {
		logger.Error("No power supply answered", "target", targetName(target.host))
		return 0, fmt.Errorf("no power supply answered")
	}

	return 1, nil
}
