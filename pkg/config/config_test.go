package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/david1155/hclsemver/pkg/version"
)

func TestLoadConfig_YAML(t *testing.T) {
	yamlContent := `
modules:
  - source: "kafka-topics-module/confluent"
    versions:
      dev:
        strategy: "range"
        version: "2.0.0"
      staging:
        version: "2.0.0"
      prod:
        strategy: "exact"
        version: "2.0.0"
  - source: "another-module/example"
    strategy: "exact"
    versions:
      dev: "1.0.0"
      staging: "1.2.0"
      prod: "1.1.0"
`
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify the parsed config
	if len(config.Modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(config.Modules))
	}

	// Check first module
	m1 := config.Modules[0]
	if m1.Source != "kafka-topics-module/confluent" {
		t.Errorf("expected source 'kafka-topics-module/confluent', got %s", m1.Source)
	}

	// Check version configs
	devConfig, err := UnmarshalVersionConfig(m1.Versions["dev"])
	if err != nil {
		t.Fatalf("failed to unmarshal dev config: %v", err)
	}
	if devConfig.Strategy != version.StrategyRange {
		t.Errorf("expected dev strategy 'range', got %s", devConfig.Strategy)
	}
	if devConfig.Version != "2.0.0" {
		t.Errorf("expected dev version '2.0.0', got %s", devConfig.Version)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	jsonContent := `{
		"modules": [
			{
				"source": "kafka-topics-module/confluent",
				"strategy": "dynamic",
				"versions": {
					"dev": {
						"strategy": "range",
						"version": "2.0.0"
					},
					"staging": {
						"version": "2.0.0"
					},
					"prod": {
						"strategy": "exact",
						"version": "2.0.0"
					}
				}
			},
			{
				"source": "another-module/example",
				"strategy": "exact",
				"versions": {
					"dev": "1.0.0",
					"staging": "1.2.0",
					"prod": "1.1.0"
				}
			}
		]
	}`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte(jsonContent), 0o600); err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify the parsed config
	if len(config.Modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(config.Modules))
	}

	// Check second module
	m2 := config.Modules[1]
	if m2.Source != "another-module/example" {
		t.Errorf("expected source 'another-module/example', got %s", m2.Source)
	}

	// Check version configs
	stagingConfig, err := UnmarshalVersionConfig(m2.Versions["staging"])
	if err != nil {
		t.Fatalf("failed to unmarshal staging config: %v", err)
	}
	if stagingConfig.Version != "1.2.0" {
		t.Errorf("expected staging version '1.2.0', got %s", stagingConfig.Version)
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "invalid JSON",
			content: `{
				"modules": [
					{
						"source": "incomplete-json
					}
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid YAML",
			content: `
modules:
  - source: "test
    versions:
      dev: ">= 1.0.0"
`,
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configFile, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			_, err := LoadConfig(configFile)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	_, err := LoadConfig("nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestGetTiersFromConfig(t *testing.T) {
	config := &Config{
		Modules: []ModuleConfig{
			{
				Source: "test-module",
				Versions: map[string]interface{}{
					"dev": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
					"staging": map[string]interface{}{
						"version": "2.0.0",
					},
				},
			},
			{
				Source: "another-module",
				Versions: map[string]interface{}{
					"prod": map[string]interface{}{
						"strategy": "exact",
						"version":  "3.0.0",
					},
					"dev": "1.5.0",
				},
			},
		},
	}

	tiers := GetTiersFromConfig(config)
	expectedTiers := map[string]bool{
		"dev":     true,
		"staging": true,
		"prod":    true,
	}

	if len(tiers) != len(expectedTiers) {
		t.Errorf("Expected %d tiers, got %d", len(expectedTiers), len(tiers))
	}

	for tier := range expectedTiers {
		if !tiers[tier] {
			t.Errorf("Expected tier %s to be present", tier)
		}
	}
}

func TestGetEffectiveStrategy(t *testing.T) {
	tests := []struct {
		name         string
		moduleConfig ModuleConfig
		tier         string
		want         version.Strategy
	}{
		{
			name: "no strategies specified",
			moduleConfig: ModuleConfig{
				Source:   "test-module",
				Versions: map[string]interface{}{"dev": "1.0.0"},
			},
			tier: "dev",
			want: version.StrategyDynamic,
		},
		{
			name: "only module strategy",
			moduleConfig: ModuleConfig{
				Source:   "test-module",
				Strategy: version.StrategyExact,
				Versions: map[string]interface{}{"dev": "1.0.0"},
			},
			tier: "dev",
			want: version.StrategyExact,
		},
		{
			name: "tier-specific strategy",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"dev": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
				},
			},
			tier: "dev",
			want: version.StrategyRange,
		},
		{
			name: "wildcard strategy",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"*": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
					"dev": "1.0.0",
				},
			},
			tier: "dev",
			want: version.StrategyRange,
		},
		{
			name: "tier strategy overrides wildcard",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"*": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
					"dev": map[string]interface{}{
						"strategy": "exact",
						"version":  "1.0.0",
					},
				},
			},
			tier: "dev",
			want: version.StrategyExact,
		},
		{
			name: "wildcard overrides module strategy",
			moduleConfig: ModuleConfig{
				Source:   "test-module",
				Strategy: version.StrategyExact,
				Versions: map[string]interface{}{
					"*": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
					"dev": "1.0.0",
				},
			},
			tier: "dev",
			want: version.StrategyRange,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetEffectiveStrategy(tc.moduleConfig, tc.tier)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetEffectiveVersionConfig(t *testing.T) {
	tests := []struct {
		name         string
		moduleConfig ModuleConfig
		tier         string
		want         VersionConfig
		wantErr      bool
	}{
		{
			name: "tier-specific config",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"dev": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
				},
			},
			tier: "dev",
			want: VersionConfig{
				Strategy: version.StrategyRange,
				Version:  "1.0.0",
			},
		},
		{
			name: "fallback to wildcard",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"*": map[string]interface{}{
						"strategy": "range",
						"version":  "1.0.0",
					},
				},
			},
			tier: "dev",
			want: VersionConfig{
				Strategy: version.StrategyRange,
				Version:  "1.0.0",
			},
		},
		{
			name: "no matching config",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"prod": "1.0.0",
				},
			},
			tier:    "dev",
			wantErr: true,
		},
		{
			name: "simple version string",
			moduleConfig: ModuleConfig{
				Source: "test-module",
				Versions: map[string]interface{}{
					"dev": "1.0.0",
				},
			},
			tier: "dev",
			want: VersionConfig{
				Version: "1.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetEffectiveVersionConfig(tc.moduleConfig, tc.tier)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got.Strategy != tc.want.Strategy || got.Version != tc.want.Version {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalVersionConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    VersionConfig
		wantErr bool
	}{
		{
			name:  "string version",
			input: "1.0.0",
			want:  VersionConfig{Version: "1.0.0"},
		},
		{
			name: "object with strategy and version",
			input: map[string]interface{}{
				"strategy": "exact",
				"version":  "1.0.0",
			},
			want: VersionConfig{
				Strategy: version.StrategyExact,
				Version:  "1.0.0",
			},
		},
		{
			name: "object with only version",
			input: map[string]interface{}{
				"version": "1.0.0",
			},
			want: VersionConfig{
				Version: "1.0.0",
			},
		},
		{
			name:    "invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UnmarshalVersionConfig(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got.Strategy != tc.want.Strategy || got.Version != tc.want.Version {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
