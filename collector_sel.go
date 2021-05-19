package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"

	"github.com/soundcloud/ipmi_exporter/freeipmi"
)

const (
	SELCollectorName CollectorName = "sel"
)

var (
	selEntriesCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel", "logs_count"),
		"Current number of log entries in the SEL.",
		[]string{},
		nil,
	)

	selFreeSpaceDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "sel", "free_space_bytes"),
		"Current free space remaining for new SEL entries.",
		[]string{},
		nil,
	)
)

type SELCollector struct{}

func (c SELCollector) Name() CollectorName {
	return SELCollectorName
}

func (c SELCollector) Cmd() string {
	return "ipmi-sel"
}

func (c SELCollector) Args() []string {
	return []string{"--info"}
}

func (c SELCollector) Collect(result freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	entriesCount, err := freeipmi.GetSELInfoEntriesCount(result)
	if err != nil {
		log.Errorf("Failed to collect SEL data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	freeSpace, err := freeipmi.GetSELInfoFreeSpace(result)
	if err != nil {
		log.Errorf("Failed to collect SEL data from %s: %s", targetName(target.host), err)
		return 0, err
	}
	ch <- prometheus.MustNewConstMetric(
		selEntriesCountDesc,
		prometheus.GaugeValue,
		entriesCount,
	)
	ch <- prometheus.MustNewConstMetric(
		selFreeSpaceDesc,
		prometheus.GaugeValue,
		freeSpace,
	)
	return 1, nil
}
