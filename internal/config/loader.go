package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// configFileNames is the ordered list of config file names to search for.
var configFileNames = []string{
	"makefmt.yml",
	"makefmt.yaml",
	".makefmt.yml",
	".makefmt.yaml",
}

// Discover returns the path of the first config file found in dir,
// following the standard search order. It returns an empty string if
// no config file is found.
func Discover(dir string) string {
	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Load reads and parses a makefmt config file. If configPath is non-empty,
// that file is loaded directly. Otherwise, Load searches the current working
// directory using Discover. If no config file is found, DefaultConfig is
// returned.
//
// Partial YAML files are supported: any fields not specified in the YAML
// retain their default values.
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		configPath = Discover(wd)
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	// Start from defaults so missing YAML fields retain non-zero defaults.
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
	}

	return cfg, nil
}

// LoadFile reads and parses a config from the given path. Unlike Load, it
// does not perform discovery — the path must be provided.
func LoadFile(path string) (*Config, error) {
	return Load(path)
}
