## next

## 1.8.0 / 2024-10-23

* Added BMC watchdog collector (#176)
* Added SEL event metrics collector (#179)
* Various dependency updates

## 1.7.0 / 2023-10-18

* Update common files
* Update build
* Update golang to 1.21
* Update dependencies
* Switch to Alpine-based Docker image
* Add missing error handling
* Added chassis cooling fault and drive fault metrics
* Now, `ipmi_dcmi_power_consumption_watts` metric is not present if Power
Measurement feature is not present. Before this change - the value was zero

## 1.6.1 / 2022-06-17

* Another "I screwed up the release" release

## 1.6.0 / 2022-06-17

* Many improvements in underlying Prometheus libraries
* Make sure `ipmimonitoring` outputs the sensor state

## 1.5.2 / 2022-03-14

* Base docker images on debian/bullseye-slim
* Update common files

## 1.5.1 / 2022-02-21

* Bugfix release for the release process itself :)

## 1.5.0 / 2022-02-21

* move to prometheus-community
* new build system and (hopefully) the docker namespace
* some fan sensors that measure in "percent of maximum rotation speed" now show
  up as fans (previously generic sensors)

Thanks a lot to all the contributors and sorry for the long wait!

## 1.4.0 / 2021-06-01

* Includes a lot of refactoring under the hood
* Add ability to customize the commands executed by the collectors - see the sample config for some examples.

## 1.3.2 / 2021-02-22

* Fixes in the `bmc` collector for systems which do not support retrieving the system firmware revision (see #57)
* Fix for sensors returning multiple events formatted as a string with questionable quoting (see #62)
* Use latest go builder container for the Docker image

## 1.3.1 / 2020-10-22

* Fix #57 - that's all :slightly_smiling_face:

## 1.3.0 / 2020-07-26

* New `sm-lan-mode` collector to get the ["LAN mode" setting](https://www.supermicro.com/support/faqs/faq.cfm?faq=28159) on Supermicro BMCs (not enabled by default)
* Added "system firmware version" (i.e. the host's BIOS version) to the BMC info metric
* Update all dependencies

## 1.2.0 / 2020-04-22

* New `sel` collector to get number of SEL entries and free space
* Update all dependencies

## 1.1.0 / 2020-02-14

* Added config option for FreeIPMI workaround-flags
* Added missing documentation bits around `ipmi-chassis` usage
* Updated dependencies to latest version

## 1.0.0 / 2019-10-18

Initial release.
