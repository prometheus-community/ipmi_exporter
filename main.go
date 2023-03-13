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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	configFile = kingpin.Flag(
		"config.file",
		"Path to configuration file.",
	).String()
	executablesPath = kingpin.Flag(
		"freeipmi.path",
		"Path to FreeIPMI executables (default: rely on $PATH).",
	).String()
	webConfig = webflag.AddFlags(kingpin.CommandLine, ":9290")

	sc = &SafeConfig{
		C: &Config{},
	}
	reloadCh chan chan error

	logger log.Logger
)

func remoteIPMIHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}

	// Remote scrape will not work without some kind of config, so be pedantic about it
	module := r.URL.Query().Get("module")
	if module == "" {
		module = "default"
	}
	if !sc.HasModule(module) {
		http.Error(w, fmt.Sprintf("Unknown module %q", module), http.StatusBadRequest)
		return
	}

	level.Debug(logger).Log("msg", "Scraping target", "target", target, "module", module)

	registry := prometheus.NewRegistry()
	remoteCollector := metaCollector{target: target, module: module, config: sc}
	registry.MustRegister(remoteCollector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func updateConfiguration(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		rc := make(chan error)
		reloadCh <- rc
		if err := <-rc; err != nil {
			http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
		}
	default:
		level.Error(logger).Log("msg", "Only POST requests allowed", "url", r.URL)
		w.Header().Set("Allow", "POST")
		http.Error(w, "Only POST requests allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("ipmi_exporter"))
	kingpin.Parse()
	logger = promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Starting ipmi_exporter", "version", version.Info())

	// Bail early if the config is bad.
	if err := sc.ReloadConfig(*configFile); err != nil {
		level.Error(logger).Log("msg", "Error parsing config file", "error", err)
		os.Exit(1)
	}

	hup := make(chan os.Signal, 1)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "error", err)
				}
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "error", err)
					rc <- err
				} else {
					rc <- nil
				}
			}
		}
	}()

	localCollector := metaCollector{target: targetLocal, module: "default", config: sc}
	prometheus.MustRegister(&localCollector)

	http.Handle("/metrics", promhttp.Handler())       // Regular metrics endpoint for local IPMI metrics.
	http.HandleFunc("/ipmi", remoteIPMIHandler)       // Endpoint to do IPMI scrapes.
	http.HandleFunc("/-/reload", updateConfiguration) // Endpoint to reload configuration.

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
            <head>
            <title>IPMI Exporter</title>
            <style>
            label{
            display:inline-block;
            width:75px;
            }
            form label {
            margin: 10px;
            }
            form input {
            margin: 10px;
            }
            </style>
            </head>
            <body>
            <h1>IPMI Exporter</h1>
            <form action="/ipmi">
            <label>Target:</label> <input type="text" name="target" placeholder="X.X.X.X" value="1.2.3.4"><br>
            <input type="submit" value="Submit">
			</form>
			<p><a href="/metrics">Local metrics</a></p>
			<p><a href="/config">Config</a></p>
            </body>
            </html>`))
	})

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "HTTP listener stopped", "error", err)
		os.Exit(1)
	}
}
