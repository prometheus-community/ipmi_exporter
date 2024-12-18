# Configuration

Simply scraping the standard `/metrics` endpoint will make the exporter emit
local IPMI metrics. If the exporter is running with sufficient privileges, no
special configuration is required. See the [privileges document](privileges.md)
for more details.

For remote metrics, the general configuration pattern is that of a
[multi-target exporter][multi-target]. Please read that guide to get the general
idea about this approach.

[multi-target]: https://prometheus.io/docs/guides/multi-target-exporter/ "Understanding and using the multi-target exporter pattern - Prometheus docs"

We offer this approach as IPMI devices often provide useful information even
while the supervised host is turned off. Also, you can have a single exporter
instance probing many (possibly thousands of) IPMI devices, rather than one
exporter per IPMI device.

**NOTE:** If you are using remote metrics, but still want to get the local
process metrics from the instance, you must use a `default` module with an
empty collectors list and use other modules for the remote hosts.

## IPMI exporter

The exporter can read a configuration file by setting `config.file` (see
above). To collect local metrics, you might not even need one. For
remote metrics, it must contain at least user names and passwords for IPMI
access to all targets to be scraped. You can additionally specify the IPMI
driver type and privilege level to use (see `man 5 freeipmi.conf` for more
details and possible values).

The config file supports the notion of "modules", so that different
configurations can be re-used for groups of targets. See the section below on
how to set the module parameter in Prometheus. The special module "default" is
used in case the scrape does not request a specific module.

The configuration file also supports a blacklist of sensors, useful in case of
OEM-specific sensors that FreeIPMI cannot deal with properly or otherwise
misbehaving sensors. This applies to both local and remote metrics.

There are two commented example configuration files, see `ipmi_local.yml` for
scraping local host metrics and `ipmi_remote.yml` for scraping remote IPMI
interfaces.

## Prometheus

### Local metrics

Collecting local IPMI metrics is fairly straightforward. Simply configure your
server to scrape the default metrics endpoint on the hosts running the
exporter.

```
- job_name: ipmi
  scrape_interval: 1m
  scrape_timeout: 30s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - 10.1.2.23:9290
    - 10.1.2.24:9290
    - 10.1.2.25:9290
```

### Remote metrics

To add your IPMI targets to Prometheus, you can use any of the supported
service discovery mechanism of your choice. The following example uses the
file-based SD and should be easy to adjust to other scenarios.

Create a YAML file that contains a list of targets, e.g.:

```
---
- targets:
  - 10.1.2.23
  - 10.1.2.24
  - 10.1.2.25
  - 10.1.2.26
  - 10.1.2.27
  - 10.1.2.28
  - 10.1.2.29
  - 10.1.2.30
  labels:
    job: ipmi_exporter
```

This file needs to be stored on the Prometheus server host. Assuming that this
file is called `/srv/ipmi_exporter/targets.yml`, and the IPMI exporter is
running on a host that has the DNS name `ipmi-exporter.internal.example.com`,
add the following to your Prometheus config:

```
- job_name: ipmi
  params:
    module: ['default']
  scrape_interval: 1m
  scrape_timeout: 30s
  metrics_path: /ipmi
  scheme: http
  file_sd_configs:
  - files:
    - /srv/ipmi_exporter/targets.yml
    refresh_interval: 5m
  relabel_configs:
  - source_labels: [__address__]
    separator: ;
    regex: (.*)
    target_label: __param_target
    replacement: ${1}
    action: replace
  - source_labels: [__param_target]
    separator: ;
    regex: (.*)
    target_label: instance
    replacement: ${1}
    action: replace
  - separator: ;
    regex: .*
    target_label: __address__
    replacement: ipmi-exporter.internal.example.com:9290
    action: replace
```

This assumes that all hosts use the default module. If you are using modules in
the config file, like in the provided `ipmi_remote.yml` example config, you
will need to specify on job for each module, using the respective group of
targets.

In a more extreme case, for example if you are using different passwords on
every host, a good approach is to generate an exporter config file that uses
the target name as module names, which would allow you to have single job that
uses label replace to set the module. Leave out the `params` in the job
definition and instead add a relabel rule like this one:

```
  - source_labels: [__address__]
    separator: ;
    regex: (.*)
    target_label: __param_module
    replacement: ${1}
    action: replace
```

For more information, e.g. how to use mechanisms other than a file to discover
the list of hosts to scrape, please refer to the [Prometheus
documentation](https://prometheus.io/docs).
