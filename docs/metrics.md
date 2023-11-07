# Exported metrics

## Scrape meta data

These metrics provide data about the scrape itself:

 - `ipmi_up{collector="<NAME>"}` is `1` if the data for this collector could
   successfully be retrieved from the remote host, `0` otherwise. The following
   collectors are available and can be enabled or disabled in the config:
   - `ipmi`: collects IPMI sensor data. If it fails, sensor metrics (see below)
     will not be available
   - `dcmi`: collects DCMI data, currently only power consumption. If it fails,
     power consumption metrics (see below) will not be available
   - `bmc`: collects BMC details. If it fails, BMC info metrics (see below)
     will not be available
   - `bmc-watchdog`: collects status of the watchdog. If it fails, BMC watchdog
     metrics (see below) will not be available
   - `chassis`: collects the current chassis power state (on/off). If it fails,
     the chassis power state metric (see below) will not be available
   - `sel`: collects system event log (SEL) details. If it fails, SEL metrics
     (see below) will not be available
   - `sm-lan-mode`: collects the "LAN mode" setting in the current BMC config.
     If it fails, the LAN mode metric (see below) will not be available
 - `ipmi_scrape_duration_seconds` is the amount of time it took to retrieve the
   data

## BMC info

This metric is only provided if the `bmc` collector is enabled.

For some basic information, there is a constant metric `ipmi_bmc_info` with
value `1` and labels providing the firmware revision and manufacturer as
returned from the BMC, and the host system's firmware version (usually the BIOS
version). Example:

    ipmi_bmc_info{firmware_revision="1.66",manufacturer_id="Dell Inc. (674)",system_firmware_version="2.6.1"} 1

**Note:** some systems do not expose the system's firmware version, in which
case it will be exported as `"N/A"`.

## BMC Watchdog

These metrics are only provided if the `bmc-watchdog` collector is enabled.

The metric `ipmi_bmc_watchdog_timer_state` shows whether the watchdog timer is
currently running (1) or stopped (0).

The metric `ipmi_bmc_watchdog_timer_use_state` shows which timer use is
currently active. Per freeipmi bmc-watchdog manual there are 5 uses. This metric
will return 1 for only one of those and 0 for the rest.

    ipmi_bmc_watchdog_timer_use_state{name="BIOS FRB2"} 1
    ipmi_bmc_watchdog_timer_use_state{name="BIOS POST"} 0
    ipmi_bmc_watchdog_timer_use_state{name="OEM"} 0
    ipmi_bmc_watchdog_timer_use_state{name="OS LOAD"} 0
    ipmi_bmc_watchdog_timer_use_state{name="SMS/OS"} 0

The metric `ipmi_bmc_watchdog_logging_state` shows whether the watchdog logging
is enabled (1) or not (0). (Note: This is reversed in freeipmi where 0 enables
logging and 1 disables it)

The metric `ipmi_bmc_watchdog_timeout_action_state` shows whether watchdog will
take an action on timeout, and if so which one. Per freeipmi bmc-watchdog manual
there are 3 actions. If no action is configured it will be reported as `None`.

    ipmi_bmc_watchdog_timeout_action_state{action="Hard Reset"} 0
    ipmi_bmc_watchdog_timeout_action_state{action="None"} 0
    ipmi_bmc_watchdog_timeout_action_state{action="Power Cycle"} 1
    ipmi_bmc_watchdog_timeout_action_state{action="Power Down"} 0

The metric `ipmi_bmc_watchdog_timeout_action_state` shows whether a pre-timeout
interrupt is currently active and if so, which one. Per freeipmi bmc-watchdog
manual there are 3 interrupts. If no interrupt is configured it will be reported
as `None`.

    ipmi_bmc_watchdog_pretimeout_interrupt_state{interrupt="Messaging Interrupt"} 0
    ipmi_bmc_watchdog_pretimeout_interrupt_state{interrupt="NMI / Diagnostic Interrupt"} 0
    ipmi_bmc_watchdog_pretimeout_interrupt_state{interrupt="None"} 1
    ipmi_bmc_watchdog_pretimeout_interrupt_state{interrupt="SMI"} 0

The metric `ipmi_bmc_watchdog_pretimeout_interval_seconds` shows the current
pre-timeout interval as measured in seconds.

The metric `ipmi_bmc_watchdog_initial_countdown_seconds` shows the configured
countdown in seconds.

The metric `ipmi_bmc_watchdog_current_countdown_seconds` shows the current
countdown in seconds.


## Chassis Power State

This metric is only provided if the `chassis` collector is enabled.

The metric `ipmi_chassis_power_state` shows the current chassis power state of
the machine. The value is 1 for power on, and 0 otherwise.

## Power consumption

This metric is only provided if the `dcmi` collector is enabled.

The metric `ipmi_dcmi_power_consumption_current_watts` can be used to monitor
the live power consumption of the machine in Watts. If in doubt, this metric
should be used over any of the sensor data (see below), even if their name
might suggest that they measure the same thing. This metric has no labels.

## System event log (SEL) info

These metrics are only provided if the `sel` collector is enabled (it isn't by
default).

The metric `ipmi_sel_entries_count` contains the current number of entries in
the SEL. It is a gauge, as the SEL can be cleared at any time. This metric has
no labels.

The metric `ipmi_sel_free_space_bytes` contains the current number of free
space for new SEL entries, in bytes. This metric has no labels.

## Supermicro LAN mode setting

This metric is only provided if the `sm-lan-mode` collector is enabled (it
isn't by default).

**NOTE:** This is a vendor-specific collector, it will only work on Supermicro
hardware, possibly even only on _some_ Supermicro systems.

**NOTE:** Retrieving this setting requires setting `privilege: "admin"` in the
config.

See e.g. https://www.supermicro.com/support/faqs/faq.cfm?faq=28159

The metric `ipmi_config_lan_mode` contains the value for the current "LAN mode"
setting (see link above): `0` for "dedicated", `1` for "shared", and `2` for
"failover".

## Sensors

These metrics are only provided if the `ipmi` collector is enabled.

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

### Temperature sensors

Temperature sensors measure a temperature in degrees Celsius and their state
usually reflects the temperature going above the vendor-recommended value. For
each temperature sensor, two metrics are exported (state and value), using the
sensor ID and the sensor name as labels. Example:

    ipmi_temperature_celsius{id="18",name="Inlet Temp"} 24
    ipmi_temperature_state{id="18",name="Inlet Temp"} 0

### Fan speed sensors

Fan speed sensors measure fan speed in rotations per minute (RPM) or as a
percentage of the maximum speed, and their state usually reflects the speed
being to low, indicating the fan might be broken. For each fan speed sensor,
two metrics are exported (state and value), using the sensor ID and the
sensor name as labels. Example:

    ipmi_fan_speed_rpm{id="12",name="Fan1A"} 4560
    ipmi_fan_speed_state{id="12",name="Fan1A"} 0

or, for a percentage based fan:

    ipmi_fan_speed_ratio{id="58",name="Fan 1 DutyCycle"} 0.2195
    ipmi_fan_speed_state{id="58",name="Fan 1 DutyCycle"} 0

### Voltage sensors

Voltage sensors measure a voltage in Volts. For each voltage sensor, two
metrics are exported (state and value), using the sensor ID and the sensor name
as labels. Example:

    ipmi_voltage_state{id="2416",name="12V"} 0
    ipmi_voltage_volts{id="2416",name="12V"} 12

### Current sensors

Current sensors measure a current in Amperes. For each current sensor, two
metrics are exported (state and value), using the sensor ID and the sensor name
as labels. Example:

    ipmi_current_state{id="83",name="Current 1"} 0
    ipmi_current_amperes{id="83",name="Current 1"} 0

### Power sensors

Power sensors measure power in Watts. For each power sensor, two metrics are
exported (state and value), using the sensor ID and the sensor name as labels.
Example:

    ipmi_power_state{id="90",name="Pwr Consumption"} 0
    ipmi_power_watts{id="90",name="Pwr Consumption"} 70

Note that based on our observations, this may or may not be a reading
reflecting the actual live power consumption. We recommend using the more
explicit [power consumption metrics](#power_consumption) for this.

### Generic sensors

For all sensors that can not be classified, two generic metrics are exported,
the state and the value. However, to provide a little more context, the sensor
type is added as label (in addition to name and ID). Example:

    ipmi_sensor_state{id="139",name="Power Cable",type="Cable/Interconnect"} 0
    ipmi_sensor_value{id="139",name="Power Cable",type="Cable/Interconnect"} NaN
