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
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	SMFanCollectorName CollectorName = "sm-fan"
)

// smFanZones are the fan zones read; most Supermicro boards have these two.
var smFanZones = []uint8{0, 1}

var (
	fanModeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "config", "fan_mode"),
		"Returns configured fan mode (0=standard, 1=full, 2=optimal, 3=PUE, 4=heavy IO).",
		nil,
		nil,
	)

	fanZoneDutyDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "fan_zone", "duty_ratio"),
		"Fan duty of a fan zone as a proportion of full duty.",
		[]string{"zone"},
		nil,
	)
)

type SMFanCollector struct{}

func (c SMFanCollector) Name() CollectorName {
	return SMFanCollectorName
}

func (c SMFanCollector) Cmd() string {
	return "ipmi-raw"
}

func (c SMFanCollector) Args() []string {
	// ipmi-raw takes its requests from Input instead.
	return []string{}
}

// Input asks for the fan mode and each zone's duty in one go.
func (c SMFanCollector) Input(_ ipmiTarget) []byte {
	var b strings.Builder
	b.WriteString("0x00 0x30 0x45 0x00\n")
	for _, zone := range smFanZones {
		fmt.Fprintf(&b, "0x00 0x30 0x70 0x66 0x00 0x%02x\n", zone)
	}
	return []byte(b.String())
}

func (c SMFanCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	responses, err := freeipmi.GetRawResponses(result)
	if err != nil {
		logger.Error("Failed to collect fan data", "target", targetName(target.host), "error", err)
		return 0, err
	}
	if len(responses) != 1+len(smFanZones) {
		logger.Error("Unexpected number of raw responses", "target", targetName(target.host), "responses", len(responses))
		return 0, fmt.Errorf("unexpected number of raw responses: %d", len(responses))
	}

	mode, err := smRawByte(responses[0])
	if err != nil {
		logger.Error("Failed to collect fan mode data", "target", targetName(target.host), "error", err)
		return 0, err
	}
	ch <- prometheus.MustNewConstMetric(fanModeDesc, prometheus.GaugeValue, float64(mode))

	for i, zone := range smFanZones {
		duty, err := smRawByte(responses[1+i])
		if err != nil {
			// Not every board has every zone, so skip one that doesn't answer.
			logger.Debug("Skipping fan zone", "target", targetName(target.host), "zone", zone, "error", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(fanZoneDutyDesc, prometheus.GaugeValue, float64(duty)/100, strconv.Itoa(int(zone)))
	}

	return 1, nil
}

// smRawByte is the value in one reply, or why there isn't one.
func smRawByte(octets []string) (uint8, error) {
	if len(octets) >= 2 && octets[1] != "00" {
		return 0, fmt.Errorf("BMC refused the command with completion code %s", octets[1])
	}
	if len(octets) != 3 {
		return 0, fmt.Errorf("unexpected number of octets in raw response: %d", len(octets))
	}

	value, err := strconv.ParseUint(octets[2], 16, 8)
	if err != nil {
		return 0, fmt.Errorf("unexpected octet in raw response: %s", octets[2])
	}

	return uint8(value), nil
}
