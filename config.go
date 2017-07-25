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
	Credentials map[string]Credentials `yaml:"credentials"`

	ExcludeSensorIDs []int64 `yaml:"exclude_sensor_ids"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

// SafeConfig wraps Config for concurrency-safe operations.
type SafeConfig struct {
	sync.RWMutex
	C *Config
}

// Credentials is the Go representation of the credentials section in the yaml
// config file.
type Credentials struct {
	User     string `yaml:"user"`
	Password string `yaml:"pass"`

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
func (s *Credentials) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Credentials
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	if err := checkOverflow(s.XXX, "credentials"); err != nil {
		return err
	}
	return nil
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
		log.Errorf("Error parsing config file: %s", err)
		return err
	}

	sc.Lock()
	sc.C = c
	sc.Unlock()

	log.Infoln("Loaded config file")
	return nil
}

// CredentialsForTarget returns the Credentials for a given target, or the
// default. It is concurrency-safe.
func (sc *SafeConfig) CredentialsForTarget(target string) (Credentials, error) {
	sc.Lock()
	defer sc.Unlock()
	if credentials, ok := sc.C.Credentials[target]; ok {
		return Credentials{
			User:     credentials.User,
			Password: credentials.Password,
		}, nil
	}
	if credentials, ok := sc.C.Credentials["default"]; ok {
		return Credentials{
			User:     credentials.User,
			Password: credentials.Password,
		}, nil
	}
	return Credentials{}, fmt.Errorf("no credentials found for target %s", target)
}

// ExcludeSensorIDs returns the list of excluded sensor IDs in a
// concurrency-safe way.
func (sc *SafeConfig) ExcludeSensorIDs() []int64 {
	sc.Lock()
	defer sc.Unlock()
	return sc.C.ExcludeSensorIDs
}
