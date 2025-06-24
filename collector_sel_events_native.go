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
	selEventsCountByStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "count_by_state"),
		"Current number of log entries in the SEL by state.",
		[]string{"state"},
		nil,
	)
	selEventsCountByNameNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "count_by_name"),
		"Current number of custom log entries in the SEL by name.",
		[]string{"name"},
		nil,
	)
	selEventsLatestTimestampNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "latest_timestamp"),
		"Latest timestamp of custom log entries in the SEL by name.",
		[]string{"name"},
		nil,
	)
)

type SELEventsNativeCollector struct{}

func (c SELEventsNativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return SELEventsCollectorName
}

func (c SELEventsNativeCollector) Cmd() string {
	return ""
}

func (c SELEventsNativeCollector) Args() []string {
	return []string{}
}

func (c SELEventsNativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	selEventConfigs := target.config.SELEvents

	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)
	res, err := client.GetSELEntries(ctx, 0)
	if err != nil {
		return 0, err
	}

	selEventByStateCount := map[string]float64{}
	selEventByNameCount := map[string]float64{}
	selEventByNameTimestamp := map[string]float64{}

	// initialize sel event metrics by zero
	for _, metricConfig := range selEventConfigs {
		selEventByNameTimestamp[metricConfig.Name] = 0
		selEventByNameCount[metricConfig.Name] = 0
	}

	for _, data := range res {
		for _, metricConfig := range selEventConfigs {
			match := metricConfig.Regex.FindStringSubmatch(data.Standard.EventString())
			logger.Debug("event regex", "regex", metricConfig.RegexRaw, "input", data.Standard.EventString(), "match", match)
			if match != nil {
				var newTimestamp = float64(data.Standard.Timestamp.Unix())
				// datetime := data.Date + " " + data.Time
				// t, err := time.Parse(SELDateTimeFormat, datetime)
				// ignore errors with invalid date or time
				// NOTE: in some cases ipmi-sel can return "PostInit" in Date and Time fields
				// Example:
				// $ ipmi-sel --comma-separated-output --output-event-state --interpret-oem-data --output-oem-event-strings
				// ID,Date,Time,Name,Type,State,Event
				// 3,PostInit,PostInit,Sensor #211,Memory,Warning,Correctable memory error ; Event Data3 = 34h
				// if err != nil {
				// logger.Debug("Failed to parse time", "target", targetName(target.host), "error", err)
				// } else {
				// newTimestamp = float64(t.Unix())
				// }
				// save latest timestamp by name metrics
				if newTimestamp > selEventByNameTimestamp[metricConfig.Name] {
					selEventByNameTimestamp[metricConfig.Name] = newTimestamp
				}
				// save count by name metrics
				selEventByNameCount[metricConfig.Name]++
			}
		}
		// save count by state metrics
		state := string(data.Standard.EventSeverity())
		_, ok := selEventByStateCount[state]
		if !ok {
			selEventByStateCount[state] = 0
		}
		selEventByStateCount[state]++
	}

	for state, value := range selEventByStateCount {
		ch <- prometheus.MustNewConstMetric(
			selEventsCountByStateNativeDesc,
			prometheus.GaugeValue,
			value,
			state,
		)
	}

	for name, value := range selEventByNameCount {
		ch <- prometheus.MustNewConstMetric(
			selEventsCountByNameNativeDesc,
			prometheus.GaugeValue,
			value,
			name,
		)
		ch <- prometheus.MustNewConstMetric(
			selEventsLatestTimestampNativeDesc,
			prometheus.GaugeValue,
			selEventByNameTimestamp[name],
			name,
		)
	}
	return 1, nil
}
