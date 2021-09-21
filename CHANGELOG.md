##  v1.4.0 / 2021-06-01

* Includes a lot of refactoring under the hood
* Add ability to customize the commands executed by the collectors - see the sample config for some examples.

##  v1.3.2 / 2021-02-22

* Fixes in the `bmc` collector for systems which do not support retrieving the system firmware revision (see #57)
* Fix for sensors returning multiple events formatted as a string with questionable quoting (see #62)
* Use latest go builder container for the Docker image

##  v1.3.1 / 2020-10-22

* Fix #57 - that's all :slightly_smiling_face:

##  v1.3.0 / 2020-07-26

* New `sm-lan-mode` collector to get the ["LAN mode" setting](https://www.supermicro.com/support/faqs/faq.cfm?faq=28159) on Supermicro BMCs (not enabled by default)
* Added "system firmware version" (i.e. the host's BIOS version) to the BMC info metric
* Update all dependencies

##  v1.2.0 / 2020-04-22

* New `sel` collector to get number of SEL entries and free space
* Update all dependencies

##  v1.1.0 / 2020-02-14

* Added config option for FreeIPMI workaround-flags
* Added missing documentation bits around `ipmi-chassis` usage
* Updated dependencies to latest version

##  v1.0.0 / 2019-10-18

Initial release.
