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

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	SELEventsCollectorName CollectorName = "sel-events"
	SELDateTimeFormat      string        = "Jan-02-2006 15:04:05"
)

var (
	selEventsCountByStateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "count_by_state"),
		"Current number of log entries in the SEL by state.",
		[]string{"state", "type"},
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
	selEventsLog = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel_events", "time"),
		"Latest timestamp of custom log entries in the SEL by event.",
		[]string{"name", "type", "state", "event"},
		nil,
	)
)

type SELEventsCollector struct{}

type stateCountKey struct {
	State string
	Type  string
}

type eventTimeKey struct {
	State string
	Type  string
	Name  string
	Event string
}

func (c SELEventsCollector) Name() CollectorName {
	return SELEventsCollectorName
}

func (c SELEventsCollector) Cmd() string {
	return "ipmi-sel"
}

func (c SELEventsCollector) Args() []string {
	return []string{
		"--quiet-cache",
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
		logger.Error("Failed to collect SEL events", "target", targetName(target.host), "error", err)
		return 0, err
	}

	selEventByStateCount := map[stateCountKey]float64{}
	seleventTime := map[eventTimeKey]float64{}
	selEventByNameCount := map[string]float64{}
	selEventByNameTimestamp := map[string]float64{}

	// initialize sel event metrics by zero
	for _, metricConfig := range selEventConfigs {
		selEventByNameTimestamp[metricConfig.Name] = 0
		selEventByNameCount[metricConfig.Name] = 0
	}

	for _, data := range events {
		var newTimestamp float64 = 0
		datetime := data.Date + " " + data.Time
		t, err := time.Parse(SELDateTimeFormat, datetime)
		// ignore errors with invalid date or time
		// NOTE: in some cases ipmi-sel can return "PostInit" in Date and Time fields
		// Example:
		// $ ipmi-sel --comma-separated-output --output-event-state --interpret-oem-data --output-oem-event-strings
		// ID,Date,Time,Name,Type,State,Event
		// 3,PostInit,PostInit,Sensor #211,Memory,Warning,Correctable memory error ; Event Data3 = 34h
		if err != nil {
			logger.Debug("Failed to parse time", "target", targetName(target.host), "error", err)
		} else {
			newTimestamp = float64(t.Unix())
		}
		for _, metricConfig := range selEventConfigs {
			match := metricConfig.Regex.FindStringSubmatch(data.Event)
			if match != nil {
				// save latest timestamp by name metrics
				if newTimestamp > selEventByNameTimestamp[metricConfig.Name] {
					selEventByNameTimestamp[metricConfig.Name] = newTimestamp
				}
				selEventByNameCount[metricConfig.Name]++
			}
		}
		// save event metrics
		stateeventTimeKey := eventTimeKey{State: data.State, Type: data.Type, Event: data.Event, Name: data.Name}
		oldTimestamp, okLog := seleventTime[stateeventTimeKey]
		if !okLog || oldTimestamp < newTimestamp {
			seleventTime[stateeventTimeKey] = newTimestamp
		}
		// save count by state metrics
		stateCountKey := stateCountKey{State: data.State, Type: data.Type}
		_, ok := selEventByStateCount[stateCountKey]
		if !ok {
			selEventByStateCount[stateCountKey] = 0
		}
		selEventByStateCount[stateCountKey]++
	}
	for stateCount, value := range selEventByStateCount {
		ch <- prometheus.MustNewConstMetric(
			selEventsCountByStateDesc,
			prometheus.GaugeValue,
			value,
			stateCount.State, stateCount.Type,
		)
	}
	for eventTime, value := range seleventTime {
		ch <- prometheus.MustNewConstMetric(
			selEventsLog,
			prometheus.GaugeValue,
			value,
			eventTime.Name, eventTime.Type, eventTime.State, eventTime.Event,
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
