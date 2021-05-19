package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/soundcloud/ipmi_exporter/freeipmi"
)

const (
	DCMICollectorName CollectorName = "dcmi"
)

var (
	powerConsumptionDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "dcmi", "power_consumption_watts"),
		"Current power consumption in Watts.",
		[]string{},
		nil,
	)
)

type DCMICollector struct{}

func (c DCMICollector) Name() CollectorName {
	return DCMICollectorName
}

func (c DCMICollector) Cmd() string {
	return "ipmi-dcmi"
}

func (c DCMICollector) Args() []string {
	return []string{"--get-system-power-statistics"}
}

func (c DCMICollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	currentPowerConsumption, err := freeipmi.GetCurrentPowerConsumption(result)
	if err != nil {
		log.Errorf("Failed to collect DCMI data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	ch <- prometheus.MustNewConstMetric(
		powerConsumptionDesc,
		prometheus.GaugeValue,
		currentPowerConsumption,
	)
	return 1, nil
}
