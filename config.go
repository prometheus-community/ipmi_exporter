package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/prometheus/common/log"
	yaml "gopkg.in/yaml.v2"
)

// Config is the Go representation of the yaml config file.
type Config struct {
	Modules map[string]IPMIConfig `yaml:"modules"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// SafeConfig wraps Config for concurrency-safe operations.
type SafeConfig struct {
	sync.RWMutex
	C *Config
}

// IPMIConfig is the Go representation of a module configuration in the yaml
// config file.
type IPMIConfig struct {
	User             string   `yaml:"user"`
	Password         string   `yaml:"pass"`
	Privilege        string   `yaml:"privilege"`
	Driver           string   `yaml:"driver"`
	Timeout          uint32   `yaml:"timeout"`
	Collectors       []string `yaml:"collectors"`
	ExcludeSensorIDs []int64  `yaml:"exclude_sensor_ids"`
	WorkaroundFlags  []string `yaml:"workaround_flags"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

var emptyConfig = IPMIConfig{Collectors: []string{"ipmi", "dcmi", "bmc", "bmc-device-id", "chassis"}}

// CollectorName is used for unmarshaling the list of collectors in the yaml config file
type CollectorName string

func checkOverflow(m map[string]interface{}, ctx string) error {
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		return fmt.Errorf("unknown fields in %s: %s", ctx, strings.Join(keys, ", "))
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if err := checkOverflow(s.XXX, "config"); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *IPMIConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*s = emptyConfig
	type plain IPMIConfig
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if err := checkOverflow(s.XXX, "modules"); err != nil {
		return err
	}

	usesBmc := false
	for _, c := range s.Collectors {
		if !(c == "ipmi" || c == "sm-lan-mode" || c == "dcmi" || c == "bmc" || c == "bmc-device-id" || c == "chassis" || c == "sel") {
			return fmt.Errorf("unknown collector name: %s", c)
		}

		if c == "bmc" || c == "bmc-device-id" {
			if !usesBmc {
				usesBmc = true
			} else {
				return fmt.Errorf("cannot use 'bmc' and 'bmc-device-id' collectors at the same time")
			}
		}
	}
	return nil
}

// ReloadConfig reloads the config in a concurrency-safe way. If the configFile
// is unreadable or unparsable, an error is returned and the old config is kept.
func (sc *SafeConfig) ReloadConfig(configFile string) error {
	var c = &Config{}
	var config []byte
	var err error

	if configFile != "" {
		config, err = ioutil.ReadFile(configFile)
		if err != nil {
			log.Errorf("Error reading config file: %s", err)
			return err
		}
	} else {
		config = []byte("# use empty file as default")
	}

	if err = yaml.Unmarshal(config, c); err != nil {
		return err
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	if configFile != "" {
		log.Infoln("Loaded config file", configFile)
	}
	return nil
}

// HasModule returns true if a given module is configured. It is concurrency-safe.
func (sc *SafeConfig) HasModule(module string) bool {
	sc.Lock()
	defer sc.Unlock()

	_, ok := sc.C.Modules[module]
	return ok
}

// ConfigForTarget returns the config for a given target/module, or the
// default. It is concurrency-safe.
func (sc *SafeConfig) ConfigForTarget(target, module string) IPMIConfig {
	sc.Lock()
	defer sc.Unlock()

	var config IPMIConfig
	var ok = false

	if module != "default" {
		config, ok = sc.C.Modules[module]
		if !ok {
			log.Errorf("Requested module %s for target %s not found, using default", module, targetName(target))
		}
	}

	// If nothing found, fall back to defaults
	if !ok {
		config, ok = sc.C.Modules["default"]
		if !ok {
			// This is probably fine for running locally, so not making this a warning
			log.Debugf("Needed default config for target %s, but none configured, using FreeIPMI defaults", targetName(target))
			config = emptyConfig
		}
	}

	return config
}
