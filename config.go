package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/prometheus/common/log"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultDriver    = "LAN_2_0"
	defaultPrivilege = "admin"
)

// Config is the Go representation of the yaml config file.
type Config struct {
	Targets    map[string]RMCPConfig  `yaml:"targets"`
	Deprecated map[string]Credentials `yaml:"credentials"`

	ExcludeSensorIDs []int64 `yaml:"exclude_sensor_ids"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// SafeConfig wraps Config for concurrency-safe operations.
type SafeConfig struct {
	sync.RWMutex
	C *Config
}

// Credentials is used temporarily to catch formerly valid configs and
// point out that they should be changed. This will be removed eventually.
type Credentials struct{}

// RMCPConfig is the Go representation of a targets configuration in the yaml
// config file.
type RMCPConfig struct {
	User      string `yaml:"user"`
	Password  string `yaml:"pass"`
	Privilege string `yaml:"privilege"`
	Driver    string `yaml:"driver"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

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
func (s *RMCPConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain RMCPConfig
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if err := checkOverflow(s.XXX, "targets"); err != nil {
		return err
	}
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *Credentials) UnmarshalYAML(unmarshal func(interface{}) error) error {
	log.Errorf("The 'credentials' section in the config file is no longer supported")
	log.Errorf("Renaming the section to 'targets' will do, but it also supports additional features")
	log.Errorf("Please check the latest documentation at https://github.com/soundcloud/ipmi_exporter")
	return fmt.Errorf("The 'credentials' section in the config file is no longer supported")
}

// ReloadConfig reloads the config in a concurrency-safe way. If the configFile
// is unreadable or unparsable, an error is returned and the old config is kept.
func (sc *SafeConfig) ReloadConfig(configFile string) error {
	var c = &Config{}

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Errorf("Error reading config file: %s", err)
		return err
	}

	if err := yaml.Unmarshal(yamlFile, c); err != nil {
		return err
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	log.Infoln("Loaded config file")
	return nil
}

// ConfigForTarget returns the config for a given target, or the
// default. It is concurrency-safe.
func (sc *SafeConfig) ConfigForTarget(target string) (RMCPConfig, error) {
	sc.Lock()
	defer sc.Unlock()

	// Start with hardcoded defaults
	config := RMCPConfig{
		Driver:    defaultDriver,
		Privilege: defaultPrivilege,
	}
	// Apply config defaults if present
	if defaultConfig, ok := sc.C.Targets["default"]; ok {
		config.User = defaultConfig.User
		config.Password = defaultConfig.Password
		if defaultConfig.Driver != "" {
			config.Driver = defaultConfig.Driver
		}
		if defaultConfig.Privilege != "" {
			config.Privilege = defaultConfig.Privilege
		}
	}
	// Apply target-specific values if present
	if targetConfig, ok := sc.C.Targets[target]; ok {
		if targetConfig.User != "" {
			config.User = targetConfig.User
		}
		if targetConfig.Password != "" {
			config.Password = targetConfig.Password
		}
		if targetConfig.Driver != "" {
			config.Driver = targetConfig.Driver
		}
		if targetConfig.Privilege != "" {
			config.Privilege = targetConfig.Privilege
		}
	}

	if config.User == "" || config.Password == "" {
		return RMCPConfig{}, fmt.Errorf("no credentials found for target %s", target)
	}
	return config, nil
}

// ExcludeSensorIDs returns the list of excluded sensor IDs in a
// concurrency-safe way.
func (sc *SafeConfig) ExcludeSensorIDs() []int64 {
	sc.Lock()
	defer sc.Unlock()
	return sc.C.ExcludeSensorIDs
}
