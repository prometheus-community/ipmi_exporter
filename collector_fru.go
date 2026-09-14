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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	FRUCollectorName CollectorName = "fru"
)

var (
	fruInfoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "fru", "info"),
		"Constant metric with value '1' providing FRU (Field Replaceable Unit) information.",
		[]string{"product_name", "product_serial", "product_manufacturer", "product_part_number", "board_product_name", "chassis_type"},
		nil,
	)
)

type FRUCollector struct{}

func (c FRUCollector) Name() CollectorName {
	return FRUCollectorName
}

func (c FRUCollector) Cmd() string {
	return "ipmi-fru"
}

func (c FRUCollector) Args() []string {
	return []string{}
}

func (c FRUCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	productName, err := freeipmi.GetFRUProductName(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "product_name", "error", err)
		productName = "N/A"
	}
	productSerial, err := freeipmi.GetFRUProductSerial(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "product_serial", "error", err)
		productSerial = "N/A"
	}
	productManufacturer, err := freeipmi.GetFRUProductManufacturer(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "product_manufacturer", "error", err)
		productManufacturer = "N/A"
	}
	productPartNumber, err := freeipmi.GetFRUProductPartNumber(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "product_part_number", "error", err)
		productPartNumber = "N/A"
	}
	boardProductName, err := freeipmi.GetFRUBoardProductName(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "board_product_name", "error", err)
		boardProductName = "N/A"
	}
	chassisType, err := freeipmi.GetFRUChassisType(result)
	if err != nil {
		logger.Debug("Failed to parse FRU data", "target", targetName(target.host), "field", "chassis_type", "error", err)
		chassisType = "N/A"
	}

	ch <- prometheus.MustNewConstMetric(
		fruInfoDesc,
		prometheus.GaugeValue,
		1,
		productName, productSerial, productManufacturer, productPartNumber, boardProductName, chassisType,
	)
	return 1, nil
}
