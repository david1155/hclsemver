package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/david1155/hclsemver/pkg/version"
	"gopkg.in/yaml.v3"
)

type VersionConfig struct {
	Strategy version.Strategy `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Version  string           `json:"version,omitempty" yaml:"version,omitempty"`
	Force    *bool            `json:"force,omitempty" yaml:"force,omitempty"`
}

type ModuleConfig struct {
	Source   string                 `json:"source" yaml:"source"`
	Strategy version.Strategy       `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Force    bool                   `json:"force,omitempty" yaml:"force,omitempty"`
	Versions map[string]interface{} `json:"versions" yaml:"versions"` // tier -> version or VersionConfig
}

type Config struct {
	Modules []ModuleConfig `json:"modules" yaml:"modules"`
}

// UnmarshalVersionConfig handles both string and object version configurations
func UnmarshalVersionConfig(data interface{}) (VersionConfig, error) {
	switch v := data.(type) {
	case string:
		return VersionConfig{Version: v}, nil
	case map[string]interface{}:
		var config VersionConfig
		if strategy, ok := v["strategy"].(string); ok {
			config.Strategy = version.Strategy(strategy)
		}
		if version, ok := v["version"].(string); ok {
			config.Version = version
		}
		if force, ok := v["force"].(bool); ok {
			config.Force = &force
		}
		return config, nil
	default:
		return VersionConfig{}, fmt.Errorf("invalid version config type: %T", data)
	}
}

// GetEffectiveVersionConfig returns the effective version configuration for a tier,
// considering wildcards and module defaults
func GetEffectiveVersionConfig(moduleConfig ModuleConfig, tier string) (VersionConfig, error) {
	// Try to get tier-specific config
	if versionData, ok := moduleConfig.Versions[tier]; ok {
		return UnmarshalVersionConfig(versionData)
	}

	// Try to get wildcard config
	if versionData, ok := moduleConfig.Versions["*"]; ok {
		return UnmarshalVersionConfig(versionData)
	}

	return VersionConfig{}, fmt.Errorf("no version configuration found for tier %s", tier)
}

// GetEffectiveStrategy returns the effective strategy for a tier, considering wildcards and module defaults
func GetEffectiveStrategy(moduleConfig ModuleConfig, tier string) version.Strategy {
	// Try to get tier-specific config
	if versionData, ok := moduleConfig.Versions[tier]; ok {
		if config, err := UnmarshalVersionConfig(versionData); err == nil && config.Strategy != "" {
			return config.Strategy
		}
	}

	// Try to get wildcard config
	if versionData, ok := moduleConfig.Versions["*"]; ok {
		if config, err := UnmarshalVersionConfig(versionData); err == nil && config.Strategy != "" {
			return config.Strategy
		}
	}

	// Fall back to module-level strategy
	if moduleConfig.Strategy != "" {
		return moduleConfig.Strategy
	}

	// Default to dynamic strategy
	return version.StrategyDynamic
}

// GetEffectiveForce returns the effective force setting for a tier,
// considering tier-specific config, wildcard config, and module defaults
func GetEffectiveForce(moduleConfig ModuleConfig, tier string) bool {
	// Try to get tier-specific config
	if versionData, ok := moduleConfig.Versions[tier]; ok {
		if config, err := UnmarshalVersionConfig(versionData); err == nil && config.Force != nil {
			return *config.Force
		}
	}

	// Try to get wildcard config
	if versionData, ok := moduleConfig.Versions["*"]; ok {
		if config, err := UnmarshalVersionConfig(versionData); err == nil && config.Force != nil {
			return *config.Force
		}
	}

	// Fall back to module-level force
	return moduleConfig.Force
}

// LoadConfig loads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty config file")
	}

	var config Config

	// Try JSON first, then YAML if that fails
	if err := json.Unmarshal(data, &config); err != nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	return &config, nil
}

// GetTiersFromConfig returns all unique tiers mentioned in the config
func GetTiersFromConfig(config *Config) map[string]bool {
	tiers := make(map[string]bool)
	for _, module := range config.Modules {
		for tier := range module.Versions {
			tiers[tier] = true
		}
	}
	return tiers
}
