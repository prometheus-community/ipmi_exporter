// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus-community/ipmi_exporter/freeipmi"

	yaml "gopkg.in/yaml.v2"
)

// CollectorName is used for unmarshaling the list of collectors in the yaml config file
type CollectorName string

// ConfiguredCollector wraps an existing collector implementation,
// potentially altering its default settings.
type ConfiguredCollector struct {
	collector    collector
	command      string
	default_args []string
	custom_args  []string
}

func (c ConfiguredCollector) Name() CollectorName {
	return c.collector.Name()
}

func (c ConfiguredCollector) Cmd() string {
	if c.command != "" {
		return c.command
	}
	return c.collector.Cmd()
}

func (c ConfiguredCollector) Args() []string {
	args := []string{}
	if c.custom_args != nil {
		// custom args come first, this way it is quite easy to
		// override a collector to use e.g. sudo
		args = append(args, c.custom_args...)
	}
	if c.default_args != nil {
		args = append(args, c.default_args...)
	} else {
		args = append(args, c.collector.Args()...)
	}
	return args
}

func (c ConfiguredCollector) Collect(output freeipmi.Result, ch chan<- prometheus.Metric, target ipmiTarget) (int, error) {
	return c.collector.Collect(output, ch, target)
}

func (c CollectorName) GetInstance() (collector, error) {
	// This is where a new collector would have to be "registered"
	switch c {
	case IPMICollectorName:
		return IPMICollector{}, nil
	case BMCCollectorName:
		return BMCCollector{}, nil
	case BMCWatchdogCollectorName:
		return BMCWatchdogCollector{}, nil
	case SELCollectorName:
		return SELCollector{}, nil
	case DCMICollectorName:
		return DCMICollector{}, nil
	case ChassisCollectorName:
		return ChassisCollector{}, nil
	case SMLANModeCollectorName:
		return SMLANModeCollector{}, nil
	}
	return nil, fmt.Errorf("invalid collector: %s", string(c))
}

func (c CollectorName) IsValid() error {
	_, err := c.GetInstance()
	return err
}

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
	User             string                     `yaml:"user"`
	Password         string                     `yaml:"pass"`
	Privilege        string                     `yaml:"privilege"`
	Driver           string                     `yaml:"driver"`
	Timeout          uint32                     `yaml:"timeout"`
	Collectors       []CollectorName            `yaml:"collectors"`
	ExcludeSensorIDs []int64                    `yaml:"exclude_sensor_ids"`
	WorkaroundFlags  []string                   `yaml:"workaround_flags"`
	CollectorCmd     map[CollectorName]string   `yaml:"collector_cmd"`
	CollectorArgs    map[CollectorName][]string `yaml:"default_args"`
	CustomArgs       map[CollectorName][]string `yaml:"custom_args"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

var defaultConfig = IPMIConfig{
	Collectors: []CollectorName{IPMICollectorName, DCMICollectorName, BMCCollectorName, ChassisCollectorName},
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
func (s *IPMIConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*s = defaultConfig
	type plain IPMIConfig
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if err := checkOverflow(s.XXX, "modules"); err != nil {
		return err
	}
	for _, c := range s.Collectors {
		if err := c.IsValid(); err != nil {
			return err
		}
	}
	return nil
}

func (c IPMIConfig) GetCollectors() []collector {
	result := []collector{}
	for _, co := range c.Collectors {
		// At this point validity has already been checked
		i, _ := co.GetInstance()
		cc := ConfiguredCollector{
			collector:    i,
			command:      c.CollectorCmd[i.Name()],
			default_args: c.CollectorArgs[i.Name()],
			custom_args:  c.CustomArgs[i.Name()],
		}
		result = append(result, cc)
	}
	return result
}

func (c IPMIConfig) GetFreeipmiConfig() string {
	var b strings.Builder
	if c.Driver != "" {
		fmt.Fprintf(&b, "driver-type %s\n", c.Driver)
	}
	if c.Privilege != "" {
		fmt.Fprintf(&b, "privilege-level %s\n", c.Privilege)
	}
	if c.User != "" {
		fmt.Fprintf(&b, "username %s\n", c.User)
	}
	if c.Password != "" {
		fmt.Fprintf(&b, "password %s\n", freeipmi.EscapePassword(c.Password))
	}
	if c.Timeout != 0 {
		fmt.Fprintf(&b, "session-timeout %d\n", c.Timeout)
	}
	if len(c.WorkaroundFlags) > 0 {
		fmt.Fprintf(&b, "workaround-flags")
		for _, flag := range c.WorkaroundFlags {
			fmt.Fprintf(&b, " %s", flag)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

// ReloadConfig reloads the config in a concurrency-safe way. If the configFile
// is unreadable or unparsable, an error is returned and the old config is kept.
func (sc *SafeConfig) ReloadConfig(configFile string) error {
	var c = &Config{}
	var config []byte
	var err error

	if configFile != "" {
		config, err = os.ReadFile(configFile)
		if err != nil {
			level.Error(logger).Log("msg", "Error reading config file", "error", err)
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
		level.Info(logger).Log("msg", "Loaded config file", "path", configFile)
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
			level.Error(logger).Log("msg", "Requested module not found, using default", "module", module, "target", targetName(target))
		}
	}

	// If nothing found, fall back to defaults
	if !ok {
		config, ok = sc.C.Modules["default"]
		if !ok {
			// This is probably fine for running locally, so not making this a warning
			level.Debug(logger).Log("msg", "Needed default config for, but none configured, using FreeIPMI defaults", "target", targetName(target))
			config = defaultConfig
		}
	}

	return config
}
