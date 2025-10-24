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
	"context"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bougou/go-ipmi"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"
)

const (
	namespace   = "ipmi"
	targetLocal = ""
)

type collector interface {
	Name() CollectorName
	Cmd() string
	Args() []string
	Collect(output freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error)
}

type metaCollector struct {
	target string
	module string
	config *SafeConfig
}

type ipmiTarget struct {
	host   string
	config IPMIConfig
}

var (
	upDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"'1' if a scrape of the IPMI device was successful, '0' otherwise.",
		[]string{"collector"},
		nil,
	)

	durationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape_duration", "seconds"),
		"Returns how long the scrape took to complete in seconds.",
		nil,
		nil,
	)
)

// Describe implements Prometheus.Collector.
func (c metaCollector) Describe(_ chan<- *prometheus.Desc) {
	// all metrics are described ad-hoc
}

func markCollectorUp(ch chan<- prometheus.Metric, name string, up int) {
	ch <- prometheus.MustNewConstMetric(
		upDesc,
		prometheus.GaugeValue,
		float64(up),
		name,
	)
}

// Collect implements Prometheus.Collector.
func (c metaCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		logger.Debug("Scrape duration", "target", targetName(c.target), "duration", duration)
		ch <- prometheus.MustNewConstMetric(
			durationDesc,
			prometheus.GaugeValue,
			duration,
		)
	}()

	config := c.config.ConfigForTarget(c.target, c.module)
	target := ipmiTarget{
		host:   c.target,
		config: config,
	}

	for _, collector := range config.GetCollectors() {
		var up int
		logger.Debug("Running collector", "target", target.host, "collector", collector.Name())

		fqcmd := collector.Cmd()
		result := freeipmi.Result{}

		// Go-native collectors return empty string as command
		if fqcmd != "" {
			if !path.IsAbs(fqcmd) {
				fqcmd = path.Join(*executablesPath, collector.Cmd())
			}
			args := collector.Args()
			cfg := config.GetFreeipmiConfig()

			result = freeipmi.Execute(fqcmd, args, cfg, target.host, logger)
		}

		up, err := collector.Collect(result, ch, target)
		if err != nil {
			logger.Error("Collector failed", "name", collector.Name(), "error", err)
		}
		markCollectorUp(ch, string(collector.Name()), up)
	}
}

func targetName(target string) string {
	if target == targetLocal {
		return "[local]"
	}
	return target
}

func NewNativeClient(ctx context.Context, target ipmiTarget) (*ipmi.Client, error) {
	var client *ipmi.Client
	var err error

	if target.host == targetLocal {
		client, err = ipmi.NewOpenClient()
	} else {
		var (
			host, port string
			p          uint64
		)
		if host, port, err = net.SplitHostPort(target.host); err != nil {
			host, port = target.host, "623"
		}
		logger.Debug("Connecting to", "host", host, "port", port)
		p, err = strconv.ParseUint(port, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid port '%s': %s", port, err.Error())
		}
		client, err = ipmi.NewClient(host, int(p), target.config.User, target.config.Password)
	}
	if err != nil {
		logger.Error("Error creating IPMI client", "target", target.host, "error", err)
		return nil, err
	}
	if target.host != targetLocal {
		// TODO it's probably safe to ditch other interfaces?
		client = client.WithInterface(ipmi.InterfaceLanplus)
	}
	if target.config.Timeout != 0 {
		client = client.WithTimeout(time.Duration(target.config.Timeout * uint32(time.Millisecond)))
	}
	if target.config.Privilege != "" {
		// TODO this means different default (unspecified) for native vs. FreeIPMI (operator)
		priv := ipmi.PrivilegeLevelUnspecified
		switch strings.ToLower(target.config.Privilege) {
		case "admin":
			priv = ipmi.PrivilegeLevelAdministrator
		case "operator":
			priv = ipmi.PrivilegeLevelOperator
		case "user":
			priv = ipmi.PrivilegeLevelUser
		}
		client = client.WithMaxPrivilegeLevel(priv)
	}
	// TODO workaround-flags not used in native client
	if err := client.Connect(ctx); err != nil {
		logger.Error("Error connecting to IPMI device", "target", target.host, "error", err)
		return nil, err
	}
	return client, nil
}

func CloseNativeClient(ctx context.Context, client *ipmi.Client) {
	if closeErr := client.Close(ctx); closeErr != nil {
		logger.Warn("Failed to close IPMI client", "target", client.Host, "error", closeErr)
	}
}
