package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMainWithFlags(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
modules:
  - source: "test-module"
    versions:
      dev:
        strategy: "range"
        version: "2.0.0"
      staging:
        version: "2.0.0"
      prod:
        strategy: "exact"
        version: "2.0.0"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test cases
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "empty config flag",
			args:    []string{"-config", ""},
			wantErr: true,
		},
		{
			name:    "nonexistent config",
			args:    []string{"-config", "nonexistent.yaml"},
			wantErr: true,
		},
		{
			name:    "valid config",
			args:    []string{"-config", configPath},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := mainWithFlags(tc.args, tmpDir)
			if tc.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
