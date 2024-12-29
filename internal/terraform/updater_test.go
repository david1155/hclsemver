package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david1155/hclsemver/pkg/version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestUpdateModuleVersionInFile(t *testing.T) {
	content := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = ">= 1, < 2"  # note spaces
}
`

	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// We'll parse new version ">=2, <3"
	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(">=2.0.0, <3.0.0")
	if err != nil {
		t.Fatalf("cannot parse new version: %v", err)
	}

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2.0.0, <3.0.0", version.StrategyRange, false)
	if err != nil {
		t.Fatalf("UpdateModuleVersionInFile error: %v", err)
	}
	if !changed {
		t.Fatal("expected a change, got false")
	}
	if oldVersion != ">= 1, < 2" {
		t.Errorf("expected old version '>= 1, < 2', got '%s'", oldVersion)
	}
	if newVersion != ">=2.0.0, <3.0.0" {
		t.Errorf("expected new version '>=2.0.0, <3.0.0', got '%s'", newVersion)
	}

	updatedBytes, err := os.ReadFile(tfFile)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	updated := string(updatedBytes)

	if !strings.Contains(updated, `version = ">=2.0.0, <3.0.0"`) {
		t.Errorf("Expected updated to >=2.0.0, <3.0.0. Got:\n%s", updated)
	}

	// Verify valid HCL
	_, diags := hclwrite.ParseConfig(updatedBytes, tfFile, hcl.InitialPos)
	if diags.HasErrors() {
		t.Errorf("Updated file is not valid HCL: %s", diags.Error())
	}
}

func TestUpdateModuleVersionInFile_NoMatch(t *testing.T) {
	content := `
module "example_module" {
  source  = "some-other-source"
  version = ">=1,<2"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(">=2,<3")
	if err != nil {
		t.Fatalf("cannot parse new version: %v", err)
	}

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2,<3", version.StrategyDynamic, false)
	if err != nil {
		t.Fatalf("UpdateModuleVersionInFile error: %v", err)
	}
	if changed {
		t.Fatal("expected NO change, got true")
	}
	if oldVersion != "" || newVersion != "" {
		t.Errorf("expected empty versions for no match, got old='%s', new='%s'", oldVersion, newVersion)
	}

	// file should remain identical
	data, _ := os.ReadFile(tfFile)
	if string(data) != content {
		t.Errorf("Expected unchanged. Got:\n%s", string(data))
	}
}

func TestUpdateModuleVersionInFile_NoVersion(t *testing.T) {
	content := `
module "kafka_topics_ziworkflows_module" {
  source = "api.env0.com/kafka-topics-module/confluent"
  # no version attribute
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// new version => ">=2,<3"
	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(">=2,<3")
	if err != nil {
		t.Fatalf("cannot parse new version: %v", err)
	}

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2,<3", version.StrategyDynamic, false)
	if err != nil {
		t.Fatalf("UpdateModuleVersionInFile error: %v", err)
	}
	if !changed {
		t.Fatal("expected a change, got false")
	}
	if oldVersion != "" {
		t.Errorf("expected empty old version, got '%s'", oldVersion)
	}
	if newVersion != ">=2,<3" {
		t.Errorf("expected new version '>=2,<3', got '%s'", newVersion)
	}

	data, _ := os.ReadFile(tfFile)
	updated := string(data)

	if !strings.Contains(updated, `version = ">=2,<3"`) {
		t.Errorf("Expected new version attribute. Got:\n%s", updated)
	}
}

func TestUpdateModuleVersionInFile_InvalidVersion(t *testing.T) {
	content := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = "??? WHAT ???"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(">=2.0.0, <3.0.0")
	if err != nil {
		t.Fatalf("cannot parse new version: %v", err)
	}

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2.0.0, <3.0.0", version.StrategyRange, false)
	if err != nil {
		t.Fatalf("UpdateModuleVersionInFile error: %v", err)
	}
	if !changed {
		t.Fatal("expected a change, got false")
	}
	if oldVersion != "??? WHAT ???" {
		t.Errorf("expected old version '??? WHAT ???', got '%s'", oldVersion)
	}
	if newVersion != ">=2.0.0, <3.0.0" {
		t.Errorf("expected new version '>=2.0.0, <3.0.0', got '%s'", newVersion)
	}

	data, _ := os.ReadFile(tfFile)
	updated := string(data)

	if !strings.Contains(updated, `version = ">=2.0.0, <3.0.0"`) {
		t.Errorf("Expected version replaced with >=2.0.0, <3.0.0. Got:\n%s", updated)
	}
}

func TestUpdateModuleVersionInFile_InvalidHCL(t *testing.T) {
	content := `
module "test" {
  source = "test-module"
  version = "1.0.0"
  # Missing closing brace
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "invalid.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange("2.0.0")
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	_, _, _, err = UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, false)
	if err == nil {
		t.Error("Expected error for invalid HCL, got nil")
	}
}

func TestUpdateModuleVersionInFile_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	content := `
module "test" {
  source = "test-module"
  version = "1.0.0"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Remove write permissions
	if err := os.Chmod(tfFile, 0444); err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}
	defer os.Chmod(tfFile, 0644) // Restore permissions for cleanup

	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange("2.0.0")
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	_, _, _, err = UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, false)
	if err == nil {
		t.Error("Expected error for write-protected file, got nil")
	}
}

func TestShouldProcessTier(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		configTiers map[string]bool
		want        bool
	}{
		{
			name:        "no tiers configured",
			path:        "/work/any/path/file.tf",
			configTiers: map[string]bool{},
			want:        true,
		},
		{
			name: "matching tier in path",
			path: "/work/dev/module/file.tf",
			configTiers: map[string]bool{
				"dev":  true,
				"prod": true,
			},
			want: true,
		},
		{
			name: "no matching tier in path",
			path: "/work/other/module/file.tf",
			configTiers: map[string]bool{
				"dev":  true,
				"prod": true,
			},
			want: false,
		},
		{
			name: "tier as filename",
			path: "/work/module/dev.tf",
			configTiers: map[string]bool{
				"dev": true,
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldProcessTier(tc.path, tc.configTiers)
			if got != tc.want {
				t.Errorf("ShouldProcessTier(%q, %v) = %v, want %v",
					tc.path, tc.configTiers, got, tc.want)
			}
		})
	}
}

func TestUpdateModuleVersionInFile_DryRun(t *testing.T) {
	content := `
module "test_module" {
  source  = "test-module"
  version = "1.0.0"
}
`
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "test.tf")
	if err := os.WriteFile(tfFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Save original content for comparison
	originalContent := content

	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange("2.0.0")
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, true)
	if err != nil {
		t.Fatalf("UpdateModuleVersionInFile error: %v", err)
	}

	// Check that the change was detected
	if !changed {
		t.Error("Expected change to be detected in dry-run mode")
	}

	// Check versions are correct
	if oldVersion != "1.0.0" {
		t.Errorf("Expected old version '1.0.0', got '%s'", oldVersion)
	}
	if newVersion != "2.0.0" {
		t.Errorf("Expected new version '2.0.0', got '%s'", newVersion)
	}

	// Check that file was not modified
	currentContent, err := os.ReadFile(tfFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(currentContent) != originalContent {
		t.Error("File was modified in dry-run mode")
	}
}
