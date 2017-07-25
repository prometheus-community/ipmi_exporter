package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	showVersion = flag.Bool("version", false, "Print version information.")
	configFile  = flag.String(
		"config.file", "ipmi.yml",
		"Path to configuration file.",
	)
	executablesPath = flag.String(
		"path", "",
		"Path to FreeIPMI executables (default: rely on $PATH).",
	)
	listenAddress = flag.String(
		"web.listen-address", ":9290",
		"Address to listen on for web interface and telemetry.",
	)

	sc = &SafeConfig{
		C: &Config{},
	}
	reloadCh chan chan error
)

func handler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}
	log.Debugf("Scraping target '%s'", target)

	registry := prometheus.NewRegistry()
	collector := collector{target: target, config: sc}
	registry.MustRegister(collector)
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
		log.Errorf("POST method expected")
		http.Error(w, "POST method expected", 400)
	}
}

func main() {
	flag.Parse()
	log.Infoln("Starting ipmi_exporter")

	// Bail early if the config is bad.
	if err := sc.ReloadConfig(*configFile); err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	hup := make(chan os.Signal)
	reloadCh = make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
				}
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorf("Error reloading config: %s", err)
					rc <- err
				} else {
					rc <- nil
				}
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())       // Normal metrics endpoint for IPMI exporter itself.
	http.HandleFunc("/ipmi", handler)                 // Endpoint to do IPMI scrapes.
	http.HandleFunc("/-/reload", updateConfiguration) // Endpoint to reload configuration.

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
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
						<p><a href="/config">Config</a></p>
            </body>
            </html>`))
	})

	log.Infof("Listening on %s", *listenAddress)
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
