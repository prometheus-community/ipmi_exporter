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
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	PMBusCollectorName CollectorName = "pmbus"
)

// What the collector asks each power supply for.
const (
	pmbusVoutMode  = 0x20
	pmbusReadVin   = 0x88
	pmbusReadIin   = 0x89
	pmbusReadVout  = 0x8b
	pmbusReadIout  = 0x8c
	pmbusReadTemp1 = 0x8d
	pmbusReadTemp2 = 0x8e
	pmbusReadFan1  = 0x90
	pmbusReadPout  = 0x96
	pmbusReadPin   = 0x97
	pmbusMfrID     = 0x99
	pmbusMfrModel  = 0x9a
	pmbusMfrSerial = 0x9e
)

// pmbusRead is one question for a power supply and the size of its answer.
type pmbusRead struct {
	command uint8
	length  uint8
}

// pmbusReads are asked of every power supply, in this order.
var pmbusReads = []pmbusRead{
	{pmbusVoutMode, 1},
	{pmbusReadVin, 2},
	{pmbusReadIin, 2},
	{pmbusReadVout, 2},
	{pmbusReadIout, 2},
	{pmbusReadTemp1, 2},
	{pmbusReadTemp2, 2},
	{pmbusReadFan1, 2},
	{pmbusReadPout, 2},
	{pmbusReadPin, 2},
}

// pmbusInfoReads are only asked with power_supply_info on.
var pmbusInfoReads = []pmbusRead{
	{pmbusMfrID, 32},
	{pmbusMfrModel, 32},
	{pmbusMfrSerial, 32},
}

// pmbusReadsFor is what the module asks each power supply.
func pmbusReadsFor(config IPMIConfig) []pmbusRead {
	if config.PowerSupplyInfo {
		return append(slices.Clone(pmbusReads), pmbusInfoReads...)
	}
	return pmbusReads
}

var (
	psuInputWattsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "input_watts"),
		"Power a power supply draws at its input, in watts.",
		[]string{"psu"},
		nil,
	)

	psuOutputWattsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "output_watts"),
		"Power a power supply delivers at its output, in watts.",
		[]string{"psu"},
		nil,
	)

	psuInputVoltsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "input_volts"),
		"Voltage at a power supply's input.",
		[]string{"psu"},
		nil,
	)

	psuInputAmperesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "input_amperes"),
		"Current a power supply draws at its input, in amperes.",
		[]string{"psu"},
		nil,
	)

	psuOutputVoltsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "output_volts"),
		"Voltage of a power supply's main output.",
		[]string{"psu"},
		nil,
	)

	psuOutputAmperesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "output_amperes"),
		"Current a power supply delivers on its main output, in amperes.",
		[]string{"psu"},
		nil,
	)

	psuTemperatureDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "temperature_celsius"),
		"Temperature inside a power supply, in degrees Celsius.",
		[]string{"psu", "sensor"},
		nil,
	)

	psuFanSpeedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "fan_speed_rpm"),
		"Speed of a power supply's fan in rotations per minute.",
		[]string{"psu"},
		nil,
	)

	psuInfoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "psu", "info"),
		"Constant metric with value '1' naming a power supply's maker, model and serial number.",
		[]string{"psu", "manufacturer", "model", "serial"},
		nil,
	)
)

type PMBusCollector struct{}

func (c PMBusCollector) Name() CollectorName {
	return PMBusCollectorName
}

func (c PMBusCollector) Cmd() string {
	return "ipmi-raw"
}

func (c PMBusCollector) Args() []string {
	// ipmi-raw takes its requests from Input instead.
	return []string{}
}

// Input asks every power supply for its readings in one go.
func (c PMBusCollector) Input(target ipmiTarget) []byte {
	var b strings.Builder
	for _, ps := range target.config.PowerSupplies {
		for _, r := range pmbusReadsFor(target.config) {
			fmt.Fprintf(&b, "0x00 0x06 0x52 0x%02x 0x%02x 0x%02x 0x%02x\n", ps.Bus, ps.Address, r.length, r.command)
		}
	}
	return []byte(b.String())
}

func (c PMBusCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	supplies := target.config.PowerSupplies
	if len(supplies) == 0 {
		logger.Error("No power supplies configured", "target", targetName(target.host))
		return 0, fmt.Errorf("no power supplies configured")
	}

	responses, err := freeipmi.GetRawResponses(result)
	if err != nil {
		logger.Error("Failed to collect power supply data", "target", targetName(target.host), "error", err)
		return 0, err
	}
	reads := pmbusReadsFor(target.config)
	if len(responses) != len(supplies)*len(reads) {
		logger.Error("Unexpected number of raw responses", "target", targetName(target.host), "responses", len(responses))
		return 0, fmt.Errorf("unexpected number of raw responses: %d", len(responses))
	}

	answered := false
	for i, ps := range supplies {
		answers := map[uint8][]byte{}
		for j, r := range reads {
			octets := responses[i*len(reads)+j]
			if len(octets) != int(r.length)+2 || octets[1] != "00" {
				logger.Debug("Power supply did not answer", "target", targetName(target.host), "psu", pmbusName(i, ps), "command", r.command, "octets", octets)
				continue
			}
			data := make([]byte, 0, r.length)
			for _, o := range octets[2:] {
				v, err := strconv.ParseUint(o, 16, 8)
				if err != nil {
					return 0, fmt.Errorf("unexpected octet in raw response: %s", o)
				}
				data = append(data, uint8(v))
			}
			answers[r.command] = data
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

// pmbusName is a power supply's label: its name, or its place in the list.
func pmbusName(i int, ps PowerSupplyConfig) string {
	if ps.Name != "" {
		return ps.Name
	}
	return strconv.Itoa(i + 1)
}

// collectPowerSupply reports what one power supply answered, and says whether
// it answered anything.
func collectPowerSupply(ch chan<- prometheus.Metric, name string, answers map[uint8][]byte) bool {
	reported := false
	gauge := func(desc *prometheus.Desc, value float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, append([]string{name}, labels...)...)
		reported = true
	}

	for _, r := range []struct {
		command uint8
		desc    *prometheus.Desc
		labels  []string
	}{
		{pmbusReadPin, psuInputWattsDesc, nil},
		{pmbusReadPout, psuOutputWattsDesc, nil},
		{pmbusReadVin, psuInputVoltsDesc, nil},
		{pmbusReadIin, psuInputAmperesDesc, nil},
		{pmbusReadIout, psuOutputAmperesDesc, nil},
		{pmbusReadTemp1, psuTemperatureDesc, []string{"1"}},
		{pmbusReadTemp2, psuTemperatureDesc, []string{"2"}},
		{pmbusReadFan1, psuFanSpeedDesc, nil},
	} {
		b := answers[r.command]
		if len(b) != 2 {
			continue
		}
		// A reading is a two-byte word: a signed power of two in the top 5 bits,
		// a signed value in the rest.
		word := uint16(b[0]) | uint16(b[1])<<8
		exponent := int(word >> 11)
		if exponent > 15 {
			exponent -= 32
		}
		mantissa := int(word & 0x7ff)
		if mantissa > 1023 {
			mantissa -= 2048
		}
		gauge(r.desc, math.Ldexp(float64(mantissa), exponent), r.labels...)
	}

	// The output voltage is an unsigned word, scaled by a power of two the
	// supply gives separately. Supplies that scale it some other way are skipped.
	if mode, vout := answers[pmbusVoutMode], answers[pmbusReadVout]; len(mode) == 1 && len(vout) == 2 && mode[0]>>5 == 0 {
		exponent := int(mode[0] & 0x1f)
		if exponent > 15 {
			exponent -= 32
		}
		gauge(psuOutputVoltsDesc, math.Ldexp(float64(uint16(vout[0])|uint16(vout[1])<<8), exponent))
	}

	// Only with all three, so the labels don't change from scrape to scrape.
	var info []string
	for _, command := range []uint8{pmbusMfrID, pmbusMfrModel, pmbusMfrSerial} {
		// A length byte, then the text, then padding.
		b := answers[command]
		if len(b) == 0 || int(b[0]) >= len(b) {
			break
		}
		text := strings.TrimRight(string(b[1:1+int(b[0])]), " \x00")
		if strings.ContainsFunc(text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
			break
		}
		info = append(info, text)
	}
	if len(info) == 3 {
		gauge(psuInfoDesc, 1, info...)
	}

	return reported
}
