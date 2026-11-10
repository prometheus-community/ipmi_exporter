// Copyright 2026 The Prometheus Authors
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
	"math"
	"testing"

	ipmitypes "github.com/bougou/go-ipmi/pkg/types"
)

func TestDiscreteSensorState(t *testing.T) {
	const (
		generic        = ipmitypes.EventReadingTypeState
		sensorSpecific = ipmitypes.EventReadingTypeSensorSpecific
	)
	nan := math.NaN()
	tests := []struct {
		name       string
		typ        ipmitypes.EventReadingType
		sensorType ipmitypes.SensorType
		offsets    []uint8
		want       float64
	}{
		{"no events", sensorSpecific, ipmitypes.SensorTypePowerSupply, nil, 0},
		{"power supply presence", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{0}, 0},
		{"power supply failure", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{1}, 2},
		{"power supply inactive", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{7}, 1},
		{"power supply presence and failure", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{0, 1}, 2},
		{"power supply presence and inactive", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{0, 7}, 1},
		{"processor thermal trip", sensorSpecific, ipmitypes.SensorTypeProcessor, []uint8{1}, 2},
		{"processor throttled", sensorSpecific, ipmitypes.SensorTypeProcessor, []uint8{10}, 1},
		{"chassis intrusion", sensorSpecific, ipmitypes.SensorTypePhysicalSecurity, []uint8{0}, 2},
		{"processor state asserted", generic, ipmitypes.SensorTypeProcessor, []uint8{1}, 2},
		{"power supply state asserted", generic, ipmitypes.SensorTypePowerSupply, []uint8{1}, 1},
		{"physical security state asserted", generic, ipmitypes.SensorTypePhysicalSecurity, []uint8{1}, 2},
		{"state deasserted", generic, ipmitypes.SensorTypeProcessor, []uint8{0}, 0},
		{"unknown event", sensorSpecific, ipmitypes.SensorTypeWatchdog2, []uint8{14}, nan},
		{"unknown event wins", sensorSpecific, ipmitypes.SensorTypePowerSupply, []uint8{1, 14}, nan},
		{"reserved event", sensorSpecific, ipmitypes.SensorTypeWatchdog2, []uint8{4}, 0},
		{"unspecified reading type", ipmitypes.EventReadingTypeUnspecified, ipmitypes.SensorTypePowerSupply, []uint8{0}, nan},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := discreteSensorState(tt.typ, tt.sensorType, tt.offsets)
			if math.IsNaN(tt.want) != math.IsNaN(got) || (!math.IsNaN(got) && got != tt.want) {
				t.Errorf("state = %v, want %v", got, tt.want)
			}
		})
	}
}
