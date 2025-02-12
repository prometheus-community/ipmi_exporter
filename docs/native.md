# Go-native IPMI implementation

The IPMI exporter now supports using a Go-native IPMI implementation, powered
by [go-ipmi](https://github.com/bougou/go-ipmi). In doing so, the exporter can
run *without* the FreeIPMI suite of tools. This mode of operation is currently
considered experimental. But it is being actively maintained/developed, and is
likely the future of this exporter.

## Should I use it?

In general, if you have the time to spare, it would be great if you could give
this a spin and provide
[feedback](https://github.com/prometheus-community/ipmi_exporter/issues) if you
can spot anything is _not_ working in native-IPMI mode that _is_ working via
FreeIPMI (the default mode).

Besides that, the native implementation also offers some real benefits:

* No more execution of external commands
* If you are affected by #227 - this cannot happen with native IPMI
* Some collectors may require less round-trips, as the exporter has more
  control over the IPMI calls being made
* The BMC watchdog collector now works remotely
* In the future, as the native implementation matures, it might provide better
  data in certain situations

## How do I use it?

Simply run the exporter with `--native-ipmi`. But please make sure to read the
rest of this document.

## What to watch out for?

There are some subtle differences to be aware of, compared to the
FreeIPMI-based collectors:

* **All collectors:**
  * The following config items no longer have any effect:
    * `driver` (only local and `LAN_2_0` are supported, please open an issue if
      you rely on another driver)
    * `workaround_flags` (not supported by go-ipmi, please open an issue if you
      rely on this)
    * `collector_cmd`, `collector_args`, `custom_cmd` - no longer applicable,
      please see also privileges section below
  * The `privilege` config item should no longer be needed (FreeIPMI restricts
    this to "OPERATOR" by default, but go-ipmi does not)
* **ipmi collector:** sensors can now have a `state` value of `3`
  ("non-recoverable") - a value that FreeIPMI does not provide
* **chassis collector:** in the native collector, the representation changed
  from `"Current drive fault state (1=false, 0=true)."` to `"Current drive
  fault state (1=true, 0=false)."`, simply because the current representation
  is weird and will likely also be changed in a future major release; same
  thing for the fan fault state
* **bmc collector:** this needs some testing, specifically the
  `system_firmware_revision`, as not all hardware supports this

## Privileges

Since no external commands are executed in native IPMI mode, none of the `sudo`
trickery described in
[privileges](https://github.com/prometheus-community/ipmi_exporter/blob/master/docs/privileges.md)
will work anymore. Make sure the exporters runs as a user that has access to
the local IPMI device (for local scraping, for remote scraping no special
privileges should be required)

## Feedback

Please [open an
issue](https://github.com/prometheus-community/ipmi_exporter/issues) if you run
into problems using the native IPMI mode.
