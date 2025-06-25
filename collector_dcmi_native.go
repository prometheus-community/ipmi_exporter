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
	powerConsumptionNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "dcmi", "power_consumption_watts"),
		"Current power consumption in Watts.",
		[]string{},
		nil,
	)
)

type DCMINativeCollector struct{}

func (c DCMINativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return DCMICollectorName
}

func (c DCMINativeCollector) Cmd() string {
	return ""
}

func (c DCMINativeCollector) Args() []string {
	return []string{}
}

func (c DCMINativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)
	res, err := client.GetDCMIPowerReading(ctx)
	if err != nil {
		logger.Error("Failed to collect DCMI data", "target", targetName(target.host), "error", err)
		return 0, err
	}

	if res.PowerMeasurementActive {
		ch <- prometheus.MustNewConstMetric(
			powerConsumptionNativeDesc,
			prometheus.GaugeValue,
			float64(res.CurrentPower),
		)
	}
	return 1, nil
}
