# Privileges

If you are running the exporter as unprivileged user, but need to execute the
FreeIPMI tools as root (as is likely necessary to access the local IPMI
interface), you can do the following:

**NOTE:** Make sure to adapt all absolute paths to match your distro!

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
       ipmi: /usr/bin/sudo
     custom_args:
       ipmi:
       - "/usr/sbin/ipmimonitoring"
     ```
     See also the [sudo example config](../ipmi_local_sudo.yml).

Note that no elevated privileges are necessary for getting remote metrics.
