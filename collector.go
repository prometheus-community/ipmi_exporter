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
	"bytes"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	vaultlib "github.com/prometheus-community/ipmi_exporter/creds"
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
func (c metaCollector) Describe(ch chan<- *prometheus.Desc) {
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
		level.Debug(logger).Log("msg", "Scrape duration", "target", targetName(c.target), "duration", duration)
		ch <- prometheus.MustNewConstMetric(
			durationDesc,
			prometheus.GaugeValue,
			duration,
		)
	}()

	config := c.config.ConfigForTarget(c.target, c.module)

	if *isHashiCorp {
		// Ensure the vault address and token file are provided
		if *vaultAddress == "" || *tokenFile == "" {
			level.Error(logger).Log("Both --ip and --token-file are required when using HashiCorp Vault.")
		}

		// Read the token from the specified file
		token, err := os.ReadFile(*tokenFile)
		if err != nil {
			level.Error(logger).Log("msg", "Error reading Token File", "error", err)
		}

		// Convert byte slices to strings and trim whitespace
		vaultToken := string(bytes.TrimSpace(token))
		vaultClient, err := vaultlib.NewVaultClient("hashicorp", *vaultAddress, vaultToken)
		vaultlib.LogError(err)
		username, password, err := vaultClient.GetCredentials(c.target)
		if err != nil {
			fmt.Print("Error:", err)
		}
		config.User = username
		config.Password = password
	}

	target := ipmiTarget{
		host:   c.target,
		config: config,
	}

	for _, collector := range config.GetCollectors() {
		var up int
		level.Debug(logger).Log("msg", "Running collector", "target", target.host, "collector", collector.Name())

		fqcmd := collector.Cmd()
		if !path.IsAbs(fqcmd) {
			fqcmd = path.Join(*executablesPath, collector.Cmd())
		}
		args := collector.Args()
		cfg := config.GetFreeipmiConfig(target)

		result := freeipmi.Execute(fqcmd, args, cfg, target.host, logger)

		up, _ = collector.Collect(result, ch, target)
		markCollectorUp(ch, string(collector.Name()), up)
	}
}

func targetName(target string) string {
	if target == targetLocal {
		return "[local]"
	}
	return target
}
