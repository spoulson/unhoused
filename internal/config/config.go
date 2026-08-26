// Package config loads and validates the YAML configuration file passed to
// unhoused via the -c CLI flag.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile describes a single Nomad environment unhoused can monitor.
type Profile struct {
	Name                 string `yaml:"name"`
	NomadURL             string `yaml:"nomadUrl"`
	NomadToken           string `yaml:"nomadToken"`
	NodeHostnameTemplate string `yaml:"nodeHostnameTemplate"`
}

// Config is the top-level unhoused configuration file.
type Config struct {
	HTTPPublicURL          string    `yaml:"httpPublicUrl"`
	ListenPort             int       `yaml:"listenPort"`
	RefreshIntervalSeconds int       `yaml:"refreshIntervalSeconds"`
	Profiles               []Profile `yaml:"profiles"`
}

const (
	defaultHTTPPublicURL          = "http://localhost"
	defaultListenPort             = 3001
	defaultRefreshIntervalSeconds = 5
	defaultNodeHostnameTemplate   = "{node}"
)

// Load reads, parses, defaults, and validates the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.applyDefaults()

	err = cfg.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.HTTPPublicURL == "" {
		c.HTTPPublicURL = defaultHTTPPublicURL
	}
	if c.ListenPort == 0 {
		c.ListenPort = defaultListenPort
	}
	if c.RefreshIntervalSeconds == 0 {
		c.RefreshIntervalSeconds = defaultRefreshIntervalSeconds
	}

	for i := range c.Profiles {
		if c.Profiles[i].NodeHostnameTemplate == "" {
			c.Profiles[i].NodeHostnameTemplate = defaultNodeHostnameTemplate
		}
	}
}

func (c *Config) validate() error {
	if len(c.Profiles) == 0 {
		return fmt.Errorf("at least one profile is required")
	}

	seenNames := make(map[string]bool, len(c.Profiles))
	for i, p := range c.Profiles {
		if p.Name == "" {
			return fmt.Errorf("profile %d: name is required", i)
		}
		if seenNames[p.Name] {
			return fmt.Errorf("profile %d: duplicate profile name %q", i, p.Name)
		}
		seenNames[p.Name] = true

		if p.NomadURL == "" {
			return fmt.Errorf("profile %q: nomadUrl is required", p.Name)
		}

		if !strings.Contains(p.NodeHostnameTemplate, "{node}") {
			return fmt.Errorf("profile %q: nodeHostnameTemplate must contain the %q placeholder", p.Name, "{node}")
		}
	}

	return nil
}
