package main

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/soundcloud/ipmi_exporter/freeipmi"
)

const (
	SMLANModeCollectorName CollectorName = "sm-lan-mode"
)

var (
	lanModeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "config", "lan_mode"),
		"Returns configured LAN mode (0=dedicated, 1=shared, 2=failover).",
		nil,
		nil,
	)
)

type SMLANModeCollector struct{}

func (c SMLANModeCollector) Name() CollectorName {
	return SMLANModeCollectorName
}

func (c SMLANModeCollector) Cmd() string {
	return "ipmi-raw"
}

func (c SMLANModeCollector) Args() []string {
	return []string{"0x0", "0x30", "0x70", "0x0c", "0"}
}

func (c SMLANModeCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	octets, err := freeipmi.GetRawOctets(result)
	if err != nil {
		log.Errorf("Failed to collect LAN mode data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	if len(octets) != 3 {
		log.Errorf("Unexpected number of octets from %s: %+v", targetName(target.host), octets)
		return 0, fmt.Errorf("unexpected number of octects in raw response: %d", len(octets))
	}

	switch octets[2] {
	case "00", "01", "02":
		value, _ := strconv.Atoi(octets[2])
		ch <- prometheus.MustNewConstMetric(lanModeDesc, prometheus.GaugeValue, float64(value))
	default:
		log.Errorf("Unexpected lan mode status (ipmi-raw) from %s: %+v", targetName(target.host), octets[2])
		return 0, fmt.Errorf("unexpected lan mode status: %s", octets[2])
	}

	return 1, nil
}
