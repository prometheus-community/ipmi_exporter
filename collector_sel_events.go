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
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	SELEventsCollectorName CollectorName = "sel-events"
)

var (
	selEventsCountByStateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "count_by_state"),
		"Current number of log entries in the SEL by state.",
		[]string{"state"},
		nil,
	)
	selEventsCountByNameDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "count_by_name"),
		"Current number of custom log entries in the SEL by name.",
		[]string{"name"},
		nil,
	)
	selEventsLatestTimestampDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "latest_timestamp"),
		"Latest timestamp of custom log entries in the SEL by name.",
		[]string{"name"},
		nil,
	)
)

type SELEventsCollector struct{}

func (c SELEventsCollector) Name() CollectorName {
	return SELEventsCollectorName
}

func (c SELEventsCollector) Cmd() string {
	return "ipmi-sel"
}

func (c SELEventsCollector) Args() []string {
	return []string{
		"-Q",
		"--comma-separated-output",
		"--no-header-output",
		"--sdr-cache-recreate",
		"--output-event-state",
		"--interpret-oem-data",
		"--entity-sensor-names",
	}
}

func (c SELEventsCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	selEventConfigs := target.config.SELEvents

	events, err := freeipmi.GetSELEvents(result)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to collect SEL events", "target", targetName(target.host), "error", err)
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

	for _, data := range events {
		for _, metricConfig := range selEventConfigs {
			match := metricConfig.Regex.FindStringSubmatch(data.Event)
			if match != nil {
				t, err := time.Parse("Jan-02-2006 15:04:05", data.Date+" "+data.Time)
				if err != nil {
					level.Error(logger).Log("msg", "Failed to collect SEL event metrics", "target", targetName(target.host), "error", err)
					return 0, err
				}
				newTimestamp := float64(t.Unix())
				// save latest timestamp by name metrics
				if newTimestamp > selEventByNameTimestamp[metricConfig.Name] {
					selEventByNameTimestamp[metricConfig.Name] = newTimestamp
				}
				// save count by name metrics
				selEventByNameCount[metricConfig.Name]++
			}
		}
		// save count by state metrics
		_, ok := selEventByStateCount[data.State]
		if !ok {
			selEventByStateCount[data.State] = 0
		}
		selEventByStateCount[data.State]++
	}

	for state, value := range selEventByStateCount {
		ch <- prometheus.MustNewConstMetric(
			selEventsCountByStateDesc,
			prometheus.GaugeValue,
			value,
			state,
		)
	}

	for name, value := range selEventByNameCount {
		ch <- prometheus.MustNewConstMetric(
			selEventsCountByNameDesc,
			prometheus.GaugeValue,
			value,
			name,
		)
		ch <- prometheus.MustNewConstMetric(
			selEventsLatestTimestampDesc,
			prometheus.GaugeValue,
			selEventByNameTimestamp[name],
			name,
		)
	}
	return 1, nil
}
