package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/soundcloud/ipmi_exporter/freeipmi"
)

const (
	ChassisCollectorName CollectorName = "chassis"
)

var (
	chassisPowerStateDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_state"),
		"Current power state (1=on, 0=off).",
		[]string{},
		nil,
	)
)

type ChassisCollector struct{}

func (c ChassisCollector) Name() CollectorName {
	return ChassisCollectorName
}

func (c ChassisCollector) Cmd() string {
	return "ipmi-chassis"
}

func (c ChassisCollector) Args() []string {
	return []string{"--get-chassis-status"}
}

func (c ChassisCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	currentChassisPowerState, err := freeipmi.GetChassisPowerState(result)
	if err != nil {
		log.Errorf("Failed to collect chassis data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	ch <- prometheus.MustNewConstMetric(
		chassisPowerStateDesc,
		prometheus.GaugeValue,
		currentChassisPowerState,
	)
	return 1, nil
}
