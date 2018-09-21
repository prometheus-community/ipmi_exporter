Prometheus IPMI Exporter
========================

This is an IPMI exporter for [Prometheus](https://prometheus.io).

It supports both the regular `/metrics` endpoint, exposing metrics from the
host that the exporter is running on, as well as an `/ipmi` endpoint that
supports IPMI over RMCP - one exporter running on one host can be used to
monitor a large number of IPMI interfaces by passing the `target` parameter to
a scrape.

The exporter relies on tools from the
[FreeIPMI](https://www.gnu.org/software/freeipmi/) suite for the actual IPMI
implementation.

## Installation

You need a Go development environment. Then, run the following to get the 
source code and build and install the binary:

    go get github.com/soundcloud/ipmi_exporter

## Running

A minimal invocation looks like this:

    ./ipmi_exporter

Supported parameters include:

 - `web.listen-address`: the address/port to listen on (default: `":9290"`)
 - `config.file`: path to the configuration file (default: `ipmi.yml`)
 - `path`: path to the FreeIPMI executables (default: rely on `$PATH`)

Make sure you have the following tools from the
[FreeIPMI](https://www.gnu.org/software/freeipmi/) suite installed:

 - `ipmimonitoring`
 - `ipmi-dcmi`
 - `bmc-info`

## Configuration

Simply scraping the standard `/metrics` endpoint will make the exporter emit
local IPMI metrics. No special configuration is required.

For remote metrics, the general configuration pattern is similar to that of the
[blackbox exporter](https://github.com/prometheus/blackbox_exporter), i.e.
Prometheus scrapes a small number (possibly one) of IPMI exporters with a
`target` URL parameter to tell the exporter which IPMI device it should use to
retrieve the IPMI metrics. We offer this approach as IPMI devices often provide
useful information even while the supervised host is turned off.  If you are
running the exporter on a separate host anyway, it makes more sense to have
only a few of them, each probing many (possibly thousands of) IPMI devices,
rather than one exporter per IPMI device.

### IPMI exporter

The exporter requires a configuration file called `ipmi.yml` (can be
overridden, see above). To collect local metrics, an empty file is technically
sufficient.  For remote metrics, it must contain user names and passwords for
IPMI access to all targets. It supports a “default” target, which is used as
fallback if the target is not explicitly listed in the file.

The configuration file also supports a blacklist of sensors, useful in case of
OEM-specific sensors that FreeIPMI cannot deal with properly or otherwise
misbehaving sensors. This applies to both local and remote metrics.

See the included `ipmi.yml` file for an example.

### Prometheus

#### Local metrics

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

#### Remote metrics

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

This file needs to be stored on the Prometheus server host.  Assuming that this
file is called `/srv/ipmi_exporter/targets.yml`, and the IPMI exporter is
running on a host that has the DNS name `ipmi-exporter.internal.example.com`,
add the following to your Prometheus config:

```
- job_name: ipmi
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
    regex: (.*)(:80)?
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

For more information, e.g. how to use mechanisms other than a file to discover
the list of hosts to scrape, please refer to the [Prometheus
documentation](https://prometheus.io/docs).

## Exported data

### Scrape meta data

These metrics provide data about the scrape itself:

 - `ipmi_up{collector="<NAME>"}` is `1` if the data for this collector could
   successfully be retrieved from the remote host, `0` otherwise. The following
   collectors are available:
   - `ipmi`: collects IPMI sensor data. If it fails, sensor metrics (see below)
     will not be available
   - `dcmi`: collects DCMI data, currently only power consumption. If it fails,
     power consumption metrics (see below) will not be available
   - `bmc`: collects BMC details. If if fails, BMC info metrics (see below)
     will not be available
 - `ipmi_scrape_duration_seconds` is the amount of time it took to retrieve the
   data

### BMC info

For some basic information, there is a constant metric `ipmi_bmc_info` with
value `1` and labels providing the firmware revision and manufacturer as
returned from the BMC. Example:

    ipmi_bmc_info{firmware_revision="2.52",manufacturer_id="Dell Inc. (674)"} 1

### Power consumption

The metric `ipmi_dcmi_power_consumption_current_watts` can be used to monitor
the live power consumption of the machine in Watts. If in doubt, this metric
should be used over any of the sensor data (see below), even if their name
might suggest that they measure the same thing. This metric has no labels.

### Sensors

IPMI sensors in general have one or two distinct pieces of information that are
of interest: a value and/or a state. The exporter always exports both, even if
the value is NaN or the state non-sensical. This is so one can still always
find the metrics to avoid ending up in a situation where one is looking for
e.g. the value of a sensor that is in a critical state, but can't find it and
assume this to be a problem.

The state of a sensor can be one of _nominal_, _warning_, _critical_, or _N/A_,
reflected by the metric values `0`, `1`, `2`, and `NaN` respectively. Think of
this as a kind of severity.

For sensors with known semantics (i.e. units), corresponding specific metrics
are exported. For everything else, generic metrics are exported.

#### Temperature sensors

Temperature sensors measure a temperature in degrees Celsius and their state
usually reflects the temperature going above the vendor-recommended value. For
each temperature sensor, two metrics are exported (state and value), using the
sensor ID and the sensor name as labels. Example:

    ipmi_temperature_celsius{id="18",name="Inlet Temp"} 24
    ipmi_temperature_state{id="18",name="Inlet Temp"} 0

#### Fan speed sensors

Fan speed sensors measure fan speed in rotations per minute (RPM) and their
state usually reflects the speed being to low, indicating the fan might be
broken. For each fan speed sensor, two metrics are exported (state and value),
using the sensor ID and the sensor name as labels. Example:

    ipmi_fan_speed_rpm{id="12",name="Fan1A"} 4560
    ipmi_fan_speed_state{id="12",name="Fan1A"} 0

#### Voltage sensors

Voltage sensors measure a voltage in Volts. For each voltage sensor, two
metrics are exported (state and value), using the sensor ID and the sensor name
as labels. Example:

    ipmi_voltage_state{id="2416",name="12V"} 0
    ipmi_voltage_volts{id="2416",name="12V"} 12

#### Current sensors

Current sensors measure a current in Amperes. For each current sensor, two
metrics are exported (state and value), using the sensor ID and the sensor name
as labels. Example:

    ipmi_current_state{id="83",name="Current 1"} 0
    ipmi_current_amperes{id="83",name="Current 1"} 0

#### Power sensors

Power sensors measure power in Watts. For each power sensor, two metrics are
exported (state and value), using the sensor ID and the sensor name as labels.
Example:

    ipmi_power_state{id="90",name="Pwr Consumption"} 0
    ipmi_power_watts{id="90",name="Pwr Consumption"} 70

Note that based on our observations, this may or may not be a reading
reflecting the actual live power consumption. We recommend using the more
explicit [power consumption metrics](#power_consumption) for this.

#### Generic sensors

For all sensors that can not be classified, two generic metrics are exported,
the state and the value.  However, to provide a little more context, the sensor
type is added as label (in addition to name and ID). Example:

    ipmi_sensor_state{id="139",name="Power Cable",type="Cable/Interconnect"} 0
    ipmi_sensor_value{id="139",name="Power Cable",type="Cable/Interconnect"} NaN

