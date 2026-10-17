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
)

func TestSensorMetricState(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		threshold bool
		events    int
		want      float64
		wantKnown bool
		wantNaN   bool
	}{
		{name: "ok", status: "ok", threshold: true, want: 0, wantKnown: true},
		{name: "lower non-critical", status: "lnc", threshold: true, want: 1, wantKnown: true},
		{name: "upper non-critical", status: "unc", threshold: true, want: 1, wantKnown: true},
		{name: "lower critical", status: "lcr", threshold: true, want: 2, wantKnown: true},
		{name: "upper critical", status: "ucr", threshold: true, want: 2, wantKnown: true},
		{name: "lower non-recoverable", status: "lnr", threshold: true, want: 3, wantKnown: true},
		{name: "upper non-recoverable", status: "unr", threshold: true, want: 3, wantKnown: true},
		{name: "no reading", status: "N/A", threshold: true, wantKnown: true, wantNaN: true},
		{name: "unknown threshold status", status: "foo", threshold: true, wantNaN: true},
		{name: "discrete without events", status: "0x0080", want: 0, wantKnown: true},
		{name: "discrete with event", status: "0x0080", events: 1, want: 1, wantKnown: true},
		{name: "discrete with events", status: "0x0083", events: 3, want: 1, wantKnown: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, known := sensorMetricState(tt.status, tt.threshold, tt.events)
			if known != tt.wantKnown {
				t.Errorf("known = %v, want %v", known, tt.wantKnown)
			}
			if tt.wantNaN {
				if !math.IsNaN(got) {
					t.Errorf("state = %v, want NaN", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("state = %v, want %v", got, tt.want)
			}
		})
	}
}
