Prometheus IPMI Exporter
========================

[![Build Status](https://circleci.com/gh/prometheus-community/ipmi_exporter.svg?style=svg)](https://circleci.com/gh/prometheus-community/ipmi_exporter)

This is an IPMI exporter for [Prometheus][prometheus].

[prometheus]: https://prometheus.io "Prometheus homepage"

It supports both the regular `/metrics` endpoint, exposing metrics from the
host that the exporter is running on, as well as an `/ipmi` endpoint that
supports IPMI over RMCP - one exporter running on one host can be used to
monitor a large number of IPMI interfaces by passing the `target` parameter to
a scrape.

The exporter relies on tools from the [FreeIPMI][freeipmi] suite for the actual
IPMI implementation.

[freeipmi]: https://www.gnu.org/software/freeipmi/ "FreeIPMI homepage"

## Installation

For most use-cases, simply download the [the latest release][releases].

[releases]: https://github.com/prometheus-community/ipmi_exporter/releases "IPMI exporter releases on Github"

For Kubernets, you can use the community-maintained [Helm chart][helm].

[helm]: https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-ipmi-exporter "IPMI exporter Helm chart in the helm-charts Github repo"

### Building from source

You need a Go development environment. Then, simply run `make` to build the
executable:

    make

This uses the common prometheus tooling to build and run some tests.

Alternatively, you can use the standard Go tooling, which will install the
executable in `$GOPATH/bin`:

    go get github.com/prometheus-community/ipmi_exporter

### Building a Docker container

You can build a Docker container with the included `docker` make target:

    make promu
    promu crossbuild -p linux/amd64
    make docker

This will not even require Go tooling on the host. See the included [docker
compose example](docker-compose.yml) for how to use the resulting container.

### Building a RPM Package

See [how to build a RPM package](contrib/rpm/README.md).

## Running

A minimal invocation looks like this:

    ./ipmi_exporter

Supported parameters include:

 - `web.listen-address`: the address/port to listen on (default: `":9290"`)
 - `config.file`: path to the configuration file (default: none)
 - `freeipmi.path`: path to the FreeIPMI executables (default: rely on `$PATH`)

For syntax and a complete list of available parameters, run:

    ./ipmi_exporter -h

Make sure you have the following tools from the [FreeIPMI][freeipmi] suite
installed:

 - `ipmimonitoring`/`ipmi-sensors`
 - `ipmi-dcmi`
 - `ipmi-raw`
 - `bmc-info`
 - `ipmi-sel`
 - `ipmi-chassis`

### Running as unprivileged user

If you are running the exporter as unprivileged user, but need to execute the
FreeIPMI tools as root, you can do the following:

  1. Add sudoers files to permit the following commands
     ```
     ipmi-exporter ALL = NOPASSWD: /usr/sbin/ipmimonitoring,\
                                   /usr/sbin/ipmi-sensors,\
                                   /usr/sbin/ipmi-dcmi,\
                                   /usr/sbin/ipmi-raw,\
                                   /usr/sbin/bmc-info,\
                                   /usr/sbin/ipmi-chassis,\
                                   /usr/sbin/ipmi-sel
     ```
  2. In your module config, override the collector command with `sudo` for
     every collector you are using and add the actual command as custom
     argument. Example for the "ipmi" collector:
     ```yaml
     collector_cmd:
       ipmi: sudo
     custom_args:
       ipmi:
       - "ipmimonitoring"
     ```
     See the last module in the [example config](ipmi_remote.yml).

### Running in Docker

**NOTE:** you should only use Docker for remote metrics.

See [Building a Docker container](#building-a-docker-container) and the example
`docker-compose.yml`. Edit the `ipmi_remote.yml` file to configure IPMI
credentials, then run with:

    sudo docker-compose up -d

By default, the server will bind on `0.0.0.0:9290`.

## Configuration

The [configuration](docs/configuration.md) document describes both the
configuration of the IPMI exporter itself as well as providing some guidance
for configuring the Prometheus server to scrape it.

## TLS and basic authentication

The IPMI Exporter supports TLS and basic authentication.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository][toolkit].

[toolkit]: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md

## Exported data

For a description of the metrics that this exporter provides, see the
[metrics](docs/metrics.md) document.
