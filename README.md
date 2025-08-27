Prometheus IPMI Exporter
========================

[![Build Status](https://circleci.com/gh/prometheus-community/ipmi_exporter.svg?style=svg)](https://circleci.com/gh/prometheus-community/ipmi_exporter)

This is an IPMI exporter for [Prometheus][prometheus].

[prometheus]: https://prometheus.io "Prometheus homepage"

It supports both the regular `/metrics` endpoint, exposing metrics from the
host on which the exporter runs, as well as a `/ipmi` endpoint that
supports IPMI over RMCP, implementing the multi-target exporter pattern. If you
plan to use the latter, please read the guide [Understanding and using the
multi-target exporter pattern][multi-target] for an overview of that paradigm.

[multi-target]: https://prometheus.io/docs/guides/multi-target-exporter/

By default, the exporter relies on tools from the [FreeIPMI][freeipmi] suite
for the actual IPMI implementation.

[freeipmi]: https://www.gnu.org/software/freeipmi/ "FreeIPMI homepage"

There is, however, experimental support for using the Go-native [go-ipmi
library](https://github.com/bougou/go-ipmi/) instead of FreeIPMI. Feedback to
help mature this support would be greatly appreciated. Please read the [native
IPMI documentation](docs/native.md) if you are interested.

## Installation

For most use-cases, simply download the [the latest release][releases].

[releases]: https://github.com/prometheus-community/ipmi_exporter/releases "IPMI exporter releases on Github"

For Kubernetes, you can use the community-maintained [Helm chart][helm].

[helm]: https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-ipmi-exporter "IPMI exporter Helm chart in the helm-charts Github repo"

Pre-built container images are available on [dockerhub][dockerhub] and
[quay.io][quay.io].

[dockerhub]: https://hub.docker.com/r/prometheuscommunity/ipmi-exporter
[quay.io]: https://quay.io/repository/prometheuscommunity/ipmi-exporter

### Building from source

You need a Go development environment. Then, simply run `make` to build the
executable:

    make

This uses common Prometheus tooling to build the exporter and run tests.

Alternatively, you can use standard Go tooling, which will install the
executable in `$GOPATH/bin`:

    go install github.com/prometheus-community/ipmi_exporter@latest

### Building a container image

You can build a container image with the included `docker` make target:

    make promu
    promu crossbuild -p linux/amd64 -p linux/arm64
    make docker

## Running

A minimal invocation looks like this:

    ./ipmi_exporter

Supported parameters include:

 - `web.listen-address`: the address/port to listen on (default: `":9290"`)
 - `config.file`: path to the configuration file (default: none)
 - `freeipmi.path`: path to the FreeIPMI executables (default: rely on `$PATH`)

For syntax and a complete list of available parameters, run:

    ./ipmi_exporter -h

Ensure that the following tools from the [FreeIPMI][freeipmi] suite are
installed:

 - `ipmimonitoring`/`ipmi-sensors`
 - `ipmi-dcmi`
 - `ipmi-raw`
 - `bmc-info`
 - `ipmi-sel`
 - `ipmi-chassis`

When running a container image, make sure to:

 - set `config.file` to the path of the config file as seen within the container
 - expose the default TCP port (9290) or set `web.listen-address` accordingly

**NOTE:** you should use containers only when collecting remote metrics.

## Configuration

The [configuration](docs/configuration.md) document describes both the
configuration of the IPMI exporter itself as well as
configuring the Prometheus server to scrape it.

## TLS and basic authentication

The IPMI Exporter supports TLS and basic authentication.

To use TLS and/or basic authentication, you need to pass a configuration file
using the `--web.config.file` parameter. The format of the file is described
[in the exporter-toolkit repository][toolkit].

[toolkit]: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md

## Exported data

For a description of the metrics that this exporter provides, see the
[metrics](docs/metrics.md) document.

## Privileges

Collecting host-local IPMI metrics requires root privileges. See
[privileges](docs/privileges.md) document for how to avoid running the exporter
as root.
