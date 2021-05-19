package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/soundcloud/ipmi_exporter/freeipmi"
)

const (
	BMCCollectorName CollectorName = "bmc"
)

var (
	bmcInfoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "bmc", "info"),
		"Constant metric with value '1' providing details about the BMC.",
		[]string{"firmware_revision", "manufacturer_id", "system_firmware_version"},
		nil,
	)
)

type BMCCollector struct{}

func (c BMCCollector) Name() CollectorName {
	return BMCCollectorName
}

func (c BMCCollector) Cmd() string {
	return "bmc-info"
}

func (c BMCCollector) Args() []string {
	return []string{}
}

func (c BMCCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	firmwareRevision, err := freeipmi.GetBMCInfoFirmwareRevision(result)
	if err != nil {
		log.Errorf("Failed to collect BMC data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	manufacturerID, err := freeipmi.GetBMCInfoManufacturerID(result)
	if err != nil {
		log.Errorf("Failed to collect BMC data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	systemFirmwareVersion, err := freeipmi.GetBMCInfoSystemFirmwareVersion(result)
	if err != nil {
		// This one is not always available.
		log.Debugf("Failed to parse bmc-info data from %s: %s", targetName(target.host), err)
		systemFirmwareVersion = "N/A"
	}
	ch <- prometheus.MustNewConstMetric(
		bmcInfoDesc,
		prometheus.GaugeValue,
		1,
		firmwareRevision, manufacturerID, systemFirmwareVersion,
	)
	return 1, nil
}
