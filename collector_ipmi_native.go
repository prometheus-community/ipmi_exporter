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
	"fmt"
	"math"
	"slices"
	"strconv"

	ipmitypes "github.com/bougou/go-ipmi/pkg/types"
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

// discreteSensorState returns the worst state of the asserted events of a
// discrete sensor, or NaN if one of them is unknown.
func discreteSensorState(typ ipmitypes.EventReadingType, sensorType ipmitypes.SensorType, offsets []uint8) float64 {
	state := 0.0
	for _, offset := range offsets {
		s := eventState(typ, sensorType, offset)
		if math.IsNaN(s) {
			return s
		}
		state = math.Max(state, s)
	}
	return state
}

// eventStateOverrides are events that go-ipmi lacks or rates differently than
// FreeIPMI.
var eventStateOverrides = []struct {
	typ        ipmitypes.EventReadingType
	sensorType ipmitypes.SensorType
	offset     uint8
	state      float64
}{
	{ipmitypes.EventReadingTypeState, ipmitypes.SensorTypePhysicalSecurity, 1, 2},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeBaseOSBootInstallationStatus, 6, 1},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeSystemACPIPowerState, 13, 2},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeWatchdog2, 4, 0},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeWatchdog2, 5, 0},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeWatchdog2, 6, 0},
	{ipmitypes.EventReadingTypeSensorSpecific, ipmitypes.SensorTypeWatchdog2, 7, 0},
}

// eventState returns the state of an asserted event, or NaN if it is unknown.
func eventState(typ ipmitypes.EventReadingType, sensorType ipmitypes.SensorType, offset uint8) float64 {
	for _, o := range eventStateOverrides {
		if o.typ == typ && o.sensorType == sensorType && o.offset == offset {
			return o.state
		}
	}
	event := typ.EventForOffset(sensorType, offset)
	if event == nil {
		return math.NaN()
	}
	if typ != ipmitypes.EventReadingTypeSensorSpecific {
		// go-ipmi rates generic events without severity for the sensor type as info
		_, ok := event.AssertionSeverityMap[sensorType]
		_, okDefault := event.AssertionSeverityMap[ipmitypes.SensorTypeReserved]
		if !ok && !okDefault {
			return math.NaN()
		}
	}
	switch typ.EventSeverity(sensorType, ipmitypes.EventData{EventData1: offset}, ipmitypes.EventDirAssertion) {
	case ipmitypes.EventSeverityInfo, ipmitypes.EventSeverityOK:
		return 0
	case ipmitypes.EventSeverityWarning, ipmitypes.EventSeverityDegraded, ipmitypes.EventSeverityNonFatal:
		return 1
	case ipmitypes.EventSeverityCritical:
		return 2
	}
	return math.NaN()
}

func (c IPMINativeCollector) Collect(_ freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	excludeIDs := target.config.ExcludeSensorIDs
	targetHost := targetName(target.host)

	filter := func(sensor *ipmitypes.Sensor) bool {
		return !slices.Contains(excludeIDs, int64(sensor.Number))
	}

	ctx := context.TODO()
	client, err := NewNativeClient(ctx, target)
	if err != nil {
		return 0, err
	}
	defer CloseNativeClient(ctx, client)
	res, err := client.GetSensors(ctx, filter)
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
			// discrete sensors have no threshold status, go-ipmi returns a hexdump
			state = math.NaN()
			if !data.IsThreshold() {
				state = discreteSensorState(data.EventReadingType, data.SensorType, data.DiscreteActiveEvents())
			}
			if math.IsNaN(state) {
				logger.Error(
					"Unknown sensor state",
					"target", targetHost,
					"state", data.Status(),
					"sensor_id", strconv.FormatInt(int64(data.Number), 10),
				)
			}
		}

		logger.Debug("Got values", "target", targetHost, "data", fmt.Sprintf("%+v", data))

		// TODO this could be greatly improved, now that we have structured data available
		switch data.SensorUnit.BaseUnit {
		case ipmitypes.SensorUnitType_RPM:
			if data.SensorUnit.Percentage {
				collectTypedSensorNative(ch, fanSpeedRatioNativeDesc, fanSpeedStateNativeDesc, state, data, 0.01)
			} else {
				collectTypedSensorNative(ch, fanSpeedRPMNativeDesc, fanSpeedStateNativeDesc, state, data, 1.0)
			}
		case ipmitypes.SensorUnitType_DegreesC:
			collectTypedSensorNative(ch, temperatureNativeDesc, temperatureStateNativeDesc, state, data, 1.0)
		case ipmitypes.SensorUnitType_Amps:
			collectTypedSensorNative(ch, currentNativeDesc, currentStateNativeDesc, state, data, 1.0)
		case ipmitypes.SensorUnitType_Volts:
			collectTypedSensorNative(ch, voltageNativeDesc, voltageStateNativeDesc, state, data, 1.0)
		case ipmitypes.SensorUnitType_Watts:
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

func collectTypedSensorNative(ch chan<- prometheus.Metric, desc, stateDesc *prometheus.Desc, state float64, data *ipmitypes.Sensor, scale float64) {
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

func collectGenericSensorNative(ch chan<- prometheus.Metric, state float64, data *ipmitypes.Sensor) {
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
