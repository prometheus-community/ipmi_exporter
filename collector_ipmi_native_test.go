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
	"math"
	"testing"
)

func TestSensorMetricState(t *testing.T) {
	cases := []struct {
		name                 string
		status               string
		isThreshold          bool
		activeDiscreteEvents []uint8
		wantState            float64
		wantSkip             bool
	}{
		{
			name:        "threshold ok",
			status:      "ok",
			isThreshold: true,
			wantState:   0,
		},
		{
			name:        "threshold lower non-critical",
			status:      "lnc",
			isThreshold: true,
			wantState:   1,
		},
		{
			name:        "threshold upper non-critical",
			status:      "unc",
			isThreshold: true,
			wantState:   1,
		},
		{
			name:        "threshold lower critical",
			status:      "lcr",
			isThreshold: true,
			wantState:   2,
		},
		{
			name:        "threshold upper critical",
			status:      "ucr",
			isThreshold: true,
			wantState:   2,
		},
		{
			name:        "threshold lower non-recoverable",
			status:      "lnr",
			isThreshold: true,
			wantState:   3,
		},
		{
			name:        "threshold upper non-recoverable",
			status:      "unr",
			isThreshold: true,
			wantState:   3,
		},
		{
			// A sensor with no valid reading at all (NotPresent /
			// ScanningDisabled / !IsReadingValid on the underlying
			// go-ipmi Sensor) must be skipped entirely, matching
			// FreeIPMI's behavior of not reporting it as a metric --
			// not mapped to a NaN state.
			name:        "no valid reading is skipped, not reported as NaN",
			status:      "N/A",
			isThreshold: true,
			wantSkip:    true,
		},
		{
			name:        "no valid reading is skipped regardless of sensor class",
			status:      "N/A",
			isThreshold: false,
			wantSkip:    true,
		},
		{
			// Discrete sensors never produce one of the threshold
			// status strings -- Status() falls back to a raw
			// "0xNNNN" hex dump of the discrete state bytes, which
			// is what the fake status below stands in for.
			name:                 "discrete sensor with no active events is nominal",
			status:               "0x0080",
			isThreshold:          false,
			activeDiscreteEvents: nil,
			wantState:            0,
		},
		{
			name:                 "discrete sensor with an active event is warning",
			status:               "0x0080",
			isThreshold:          false,
			activeDiscreteEvents: []uint8{2},
			wantState:            1,
		},
		{
			name:                 "discrete sensor with multiple active events is still warning",
			status:               "0x0083",
			isThreshold:          false,
			activeDiscreteEvents: []uint8{0, 1, 7},
			wantState:            1,
		},
		{
			// A threshold sensor reporting a status string outside
			// the known set is genuinely unrecognized -- unlike the
			// discrete case, this has always meant "unknown" and
			// should stay that way.
			name:        "unrecognized threshold status maps to NaN",
			status:      "something-unexpected",
			isThreshold: true,
			wantState:   math.NaN(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotState, gotSkip := sensorMetricState(tc.status, tc.isThreshold, tc.activeDiscreteEvents)

			if gotSkip != tc.wantSkip {
				t.Errorf("sensorMetricState(%q, %v, %v) skip = %v, want %v",
					tc.status, tc.isThreshold, tc.activeDiscreteEvents, gotSkip, tc.wantSkip)
			}

			if tc.wantSkip {
				// state is unspecified when skip is true; nothing
				// further to assert.
				return
			}

			if math.IsNaN(tc.wantState) {
				if !math.IsNaN(gotState) {
					t.Errorf("sensorMetricState(%q, %v, %v) state = %v, want NaN",
						tc.status, tc.isThreshold, tc.activeDiscreteEvents, gotState)
				}
				return
			}

			if gotState != tc.wantState {
				t.Errorf("sensorMetricState(%q, %v, %v) state = %v, want %v",
					tc.status, tc.isThreshold, tc.activeDiscreteEvents, gotState, tc.wantState)
			}
		})
	}
}
