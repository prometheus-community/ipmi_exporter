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
	selEntriesCountNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel", "logs_count"),
		"Current number of log entries in the SEL.",
		[]string{},
		nil,
	)

	selFreeSpaceNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel", "free_space_bytes"),
		"Current free space remaining for new SEL entries.",
		[]string{},
		nil,
	)
)

type SELNativeCollector struct{}

func (c SELNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return SELCollectorName
}

func (c SELNativeCollector) Cmd() string {
	return ""
}

func (c SELNativeCollector) Args() []string {
	return []string{""}
}

func (c SELNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)
	res, err := client.GetSELInfo(ctx)
	if err != nil {
		return 0, err
	}

	ch <- prometheus.MustNewConstMetric(
		selEntriesCountNativeDesc,
		prometheus.GaugeValue,
		float64(res.Entries),
	)
	ch <- prometheus.MustNewConstMetric(
		selFreeSpaceNativeDesc,
		prometheus.GaugeValue,
		float64(res.FreeBytes),
	)
	return 1, nil
}
