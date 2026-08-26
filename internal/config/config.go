// Package config loads and validates the YAML configuration file passed to
// unhoused via the -c CLI flag.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Environment is a Nomad deployment environment.
type Environment string

const (
	EnvironmentStaging    Environment = "staging"
	EnvironmentProduction Environment = "production"
)

// Region is a supported Nomad region.
type Region string

const (
	RegionUSWest1     Region = "us-west1"
	RegionUSEast4     Region = "us-east4"
	RegionEuropeWest1 Region = "europe-west1"
	RegionAUSE1       Region = "ause1"
)

var shortRegions = map[Region]string{
	RegionUSEast4:     "use4",
	RegionUSWest1:     "usw1",
	RegionEuropeWest1: "euw1",
	RegionAUSE1:       "ause1",
}

// ShortRegion returns the abbreviated form of r used in derived hostnames.
func ShortRegion(r Region) (string, error) {
	short, ok := shortRegions[r]
	if !ok {
		return "", fmt.Errorf("unknown region %q", r)
	}
	return short, nil
}

// Profile describes a single Nomad environment unhoused can monitor.
type Profile struct {
	Name                 string      `yaml:"name"`
	Environment          Environment `yaml:"environment"`
	Region               Region      `yaml:"region"`
	NomadURL             string      `yaml:"nomadUrl"`
	NomadToken           string      `yaml:"nomadToken"`
	NodeHostnameTemplate string      `yaml:"nodeHostnameTemplate"`
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

		if p.Environment != EnvironmentStaging && p.Environment != EnvironmentProduction {
			return fmt.Errorf("profile %q: invalid environment %q", p.Name, p.Environment)
		}

		_, err := ShortRegion(p.Region)
		if err != nil {
			return fmt.Errorf("profile %q: invalid region %q", p.Name, p.Region)
		}

		if p.NomadURL == "" {
			return fmt.Errorf("profile %q: nomadUrl is required", p.Name)
		}

		if !strings.Contains(p.NodeHostnameTemplate, "{node}") {
			return fmt.Errorf("profile %q: nodeHostnameTemplate must contain the %q placeholder", p.Name, "{node}")
		}
	}

	return nil
}
