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
	"math"
	"strconv"

	"github.com/bougou/go-ipmi"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

var (
	sensorStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "state"),
		"Indicates the severity of the state reported by an IPMI sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name", "type"},
		nil,
	)

	sensorValueNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sensor", "value"),
		"Generic data read from an IPMI sensor of unknown type, relying on labels for context.",
		[]string{"id", "name", "type"},
		nil,
	)

	fanSpeedRPMNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "fan_speed", "rpm"),
		"Fan speed in rotations per minute.",
		[]string{"id", "name"},
		nil,
	)

	fanSpeedRatioNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "fan_speed", "ratio"),
		"Fan speed as a proportion of the maximum speed.",
		[]string{"id", "name"},
		nil,
	)

	fanSpeedStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "fan_speed", "state"),
		"Reported state of a fan speed sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name"},
		nil,
	)

	temperatureNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "temperature", "celsius"),
		"Temperature reading in degree Celsius.",
		[]string{"id", "name"},
		nil,
	)

	temperatureStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "temperature", "state"),
		"Reported state of a temperature sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name"},
		nil,
	)

	voltageNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "voltage", "volts"),
		"Voltage reading in Volts.",
		[]string{"id", "name"},
		nil,
	)

	voltageStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "voltage", "state"),
		"Reported state of a voltage sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name"},
		nil,
	)

	currentNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "current", "amperes"),
		"Current reading in Amperes.",
		[]string{"id", "name"},
		nil,
	)

	currentStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "current", "state"),
		"Reported state of a current sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name"},
		nil,
	)

	powerNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "power", "watts"),
		"Power reading in Watts.",
		[]string{"id", "name"},
		nil,
	)

	powerStateNativeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "power", "state"),
		"Reported state of a power sensor (0=nominal, 1=warning, 2=critical, 3=non-recoverable).",
		[]string{"id", "name"},
		nil,
	)
)

type IPMINativeCollector struct{}

func (c IPMINativeCollector) Name() CollectorName {
	// The name is intentionally the same as the non-native collector
	return IPMICollectorName
}

func (c IPMINativeCollector) Cmd() string {
	return ""
}

func (c IPMINativeCollector) Args() []string {
	return []string{}
}

func (c IPMINativeCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	excludeIds := target.config.ExcludeSensorIDs
	targetHost := targetName(target.host)

	filter := func(sensor *ipmi.Sensor) bool {
		for _, id := range excludeIds {
			if id == int64(sensor.Number) {
				return false
			}
		}
		return true
	}

	client, err := NewNativeClient(target)
	if err != nil {
		return 0, err
	}
	res, err := client.GetSensors(context.TODO(), filter)
	if err != nil {
		return 0, err
	}

	// results, err := freeipmi.GetSensorData(result, excludeIds)
	// if err != nil {
	// 	logger.Error("Failed to collect sensor data", "target", targetHost, "error", err)
	// 	return 0, err
	// }
	for _, data := range res {
		var state float64

		switch data.Status() {
		case "ok":
			state = 0
		case "lnc", "unc": // lower/upper non-critical
			state = 1
		case "lcr", "ucr": // lower/upper critical
			state = 2
		case "lnr", "unr": // lower/upper non-recoverable
			state = 3 // TODO this is new
		case "N/A":
			state = math.NaN()
		default:
			logger.Error("Unknown sensor state", "target", targetHost, "state", data.Status())
			state = math.NaN()
		}

		logger.Debug("Got values", "target", targetHost, "data", fmt.Sprintf("%+v", data))

		// TODO this could be greatly improved, now that we have structured data available
		switch data.SensorUnit.BaseUnit {
		case ipmi.SensorUnitType_RPM:
			if data.SensorUnit.Percentage {
				collectTypedSensorNative(ch, fanSpeedRatioNativeDesc, fanSpeedStateNativeDesc, state, data, 0.01)
			} else {

				collectTypedSensorNative(ch, fanSpeedRPMNativeDesc, fanSpeedStateNativeDesc, state, data, 1.0)
			}
		case ipmi.SensorUnitType_DegreesC:
			collectTypedSensorNative(ch, temperatureNativeDesc, temperatureStateNativeDesc, state, data, 1.0)
		case ipmi.SensorUnitType_Amps:
			collectTypedSensorNative(ch, currentNativeDesc, currentStateNativeDesc, state, data, 1.0)
		case ipmi.SensorUnitType_Volts:
			collectTypedSensorNative(ch, voltageNativeDesc, voltageStateNativeDesc, state, data, 1.0)
		case ipmi.SensorUnitType_Watts:
			collectTypedSensorNative(ch, powerNativeDesc, powerStateNativeDesc, state, data, 1.0)
		default:
			collectGenericSensorNative(ch, state, data)
		}
	}
	return 1, nil
}

func (c IPMINativeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sensorStateDesc
	ch <- sensorValueDesc
	ch <- fanSpeedRPMDesc
	ch <- fanSpeedRatioDesc
	ch <- fanSpeedStateDesc
	ch <- temperatureDesc
	ch <- temperatureStateDesc
	ch <- voltageDesc
	ch <- voltageStateDesc
	ch <- currentDesc
	ch <- currentStateDesc
	ch <- powerDesc
	ch <- powerStateDesc
}

func collectTypedSensorNative(ch chan<- prometheus.Metric, desc, stateDesc *prometheus.Desc, state float64, data *ipmi.Sensor, scale float64) {
	ch <- prometheus.MustNewConstMetric(
		desc,
		prometheus.GaugeValue,
		data.Value*scale,
		strconv.FormatInt(int64(data.Number), 10),
		data.Name,
	)
	ch <- prometheus.MustNewConstMetric(
		stateDesc,
		prometheus.GaugeValue,
		state,
		strconv.FormatInt(int64(data.Number), 10),
		data.Name,
	)
}

func collectGenericSensorNative(ch chan<- prometheus.Metric, state float64, data *ipmi.Sensor) {
	ch <- prometheus.MustNewConstMetric(
		sensorValueNativeDesc,
		prometheus.GaugeValue,
		data.Value,
		strconv.FormatInt(int64(data.Number), 10),
		data.Name,
		data.SensorType.String(),
	)
	ch <- prometheus.MustNewConstMetric(
		sensorStateNativeDesc,
		prometheus.GaugeValue,
		state,
		strconv.FormatInt(int64(data.Number), 10),
		data.Name,
		data.SensorType.String(),
	)
}
