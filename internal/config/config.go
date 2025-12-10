package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Target describes a destination MySQL instance.
type Target struct {
	Name string `json:"name" yaml:"name"`
	DSN  string `json:"dsn" yaml:"dsn"`
}

// FileConfig represents configuration loaded from a YAML/JSON file.
type FileConfig struct {
	Source         string   `json:"source" yaml:"source"`
	Targets        []Target `json:"targets" yaml:"targets"`
	Include        []string `json:"include" yaml:"include"`
	Exclude        []string `json:"exclude" yaml:"exclude"`
	DryRun         bool     `json:"dry_run" yaml:"dry_run"`
	DropMissing    bool     `json:"drop_missing" yaml:"drop_missing"`
	ForceOverwrite bool     `json:"force_overwrite" yaml:"force_overwrite"`
	ReportPath     string   `json:"report_path" yaml:"report_path"`
	Concurrency    int      `json:"concurrency" yaml:"concurrency"`
	Verbose        bool     `json:"verbose" yaml:"verbose"`
}

// CLIConfig captures values provided via command-line flags (which may be unset).
type CLIConfig struct {
	Source         string
	Targets        []Target
	Include        []string
	Exclude        []string
	DryRun         *bool
	DropMissing    *bool
	ForceOverwrite *bool
	ReportPath     string
	Concurrency    *int
	Verbose        *bool
}

// RuntimeConfig is the fully merged, validated configuration.
type RuntimeConfig struct {
	Source         string
	Targets        []Target
	Include        []string
	Exclude        []string
	DryRun         bool
	DropMissing    bool
	ForceOverwrite bool
	ReportPath     string
	Concurrency    int
	Verbose        bool
}

// Load reads configuration from a YAML or JSON file.
func Load(path string) (FileConfig, error) {
	if path == "" {
		return FileConfig{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("read config: %w", err)
	}
	var cfg FileConfig
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return FileConfig{}, fmt.Errorf("parse yaml: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return FileConfig{}, fmt.Errorf("parse json: %w", err)
		}
	}
	return cfg, nil
}

// Merge combines file and CLI configuration, preferring CLI when provided.
func Merge(fileCfg FileConfig, cliCfg CLIConfig) RuntimeConfig {
	out := RuntimeConfig{
		Source:         fileCfg.Source,
		Targets:        append([]Target(nil), fileCfg.Targets...),
		Include:        append([]string(nil), fileCfg.Include...),
		Exclude:        append([]string(nil), fileCfg.Exclude...),
		DryRun:         fileCfg.DryRun,
		DropMissing:    fileCfg.DropMissing,
		ForceOverwrite: fileCfg.ForceOverwrite,
		ReportPath:     fileCfg.ReportPath,
		Concurrency:    fileCfg.Concurrency,
		Verbose:        fileCfg.Verbose,
	}

	if cliCfg.Source != "" {
		out.Source = cliCfg.Source
	}
	if len(cliCfg.Targets) > 0 {
		out.Targets = cliCfg.Targets
	}
	if len(cliCfg.Include) > 0 {
		out.Include = cliCfg.Include
	}
	if len(cliCfg.Exclude) > 0 {
		out.Exclude = cliCfg.Exclude
	}
	if cliCfg.DryRun != nil {
		out.DryRun = *cliCfg.DryRun
	}
	if cliCfg.DropMissing != nil {
		out.DropMissing = *cliCfg.DropMissing
	}
	if cliCfg.ForceOverwrite != nil {
		out.ForceOverwrite = *cliCfg.ForceOverwrite
	}
	if cliCfg.ReportPath != "" {
		out.ReportPath = cliCfg.ReportPath
	}
	if cliCfg.Concurrency != nil {
		out.Concurrency = *cliCfg.Concurrency
	}
	if cliCfg.Verbose != nil {
		out.Verbose = *cliCfg.Verbose
	}
	return out
}

// Validate ensures required fields are present and fill defaults.
func (c *RuntimeConfig) Validate() error {
	if c.Source == "" {
		return errors.New("missing source DSN (flag or config)")
	}
	if len(c.Targets) == 0 {
		return errors.New("missing at least one target DSN")
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}
	return nil
}
