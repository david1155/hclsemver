package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/david1155/hclsemver/pkg/version"
)

func TestUpdateModuleVersionInFile(t *testing.T) {
	// Create a temporary directory for test files
	dir, err := os.MkdirTemp("", "TestUpdateModuleVersionInFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a test file
	testFile := filepath.Join(dir, "test.tf")
	content := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = "1.0.0"
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test updating the version
	newVersion := ">= 2.0.0, < 3.0.0"
	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(newVersion)
	if err != nil {
		t.Fatal(err)
	}

	changed, oldVersion, resultVersion, err := UpdateModuleVersionInFile(testFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, newVersion, version.StrategyRange, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected a change, got false")
	}
	if oldVersion != "1.0.0" {
		t.Errorf("expected old version '1.0.0', got '%s'", oldVersion)
	}
	if resultVersion != newVersion {
		t.Errorf("expected new version '%s', got '%s'", newVersion, resultVersion)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the version was updated correctly
	expectedContent := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = ">= 2.0.0, < 3.0.0"
}
`
	if string(updatedContent) != expectedContent {
		t.Errorf("Expected updated to %s. Got:\n%s", newVersion, string(updatedContent))
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

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2,<3", version.StrategyDynamic, false, false)
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
	tests := []struct {
		name    string
		force   bool
		wantMod bool
	}{
		{
			name:    "no force flag",
			force:   false,
			wantMod: false,
		},
		{
			name:    "with force flag",
			force:   true,
			wantMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, ">=2,<3", version.StrategyDynamic, false, tt.force)
			if err != nil {
				t.Fatalf("UpdateModuleVersionInFile error: %v", err)
			}

			if changed != tt.wantMod {
				t.Fatalf("expected changed=%v, got %v", tt.wantMod, changed)
			}

			if tt.wantMod {
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
			} else {
				// Check file wasn't modified
				data, _ := os.ReadFile(tfFile)
				if string(data) != content {
					t.Errorf("Expected file to remain unchanged. Got:\n%s", string(data))
				}
			}
		})
	}
}

func TestUpdateModuleVersionInFile_InvalidVersion(t *testing.T) {
	// Create a temporary directory for test files
	dir, err := os.MkdirTemp("", "TestUpdateModuleVersionInFile_InvalidVersion")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a test file
	testFile := filepath.Join(dir, "test.tf")
	content := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = "invalid"
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test updating the version
	newVersion := ">= 2.0.0, < 3.0.0"
	newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(newVersion)
	if err != nil {
		t.Fatal(err)
	}

	changed, oldVersion, resultVersion, err := UpdateModuleVersionInFile(testFile, "kafka-topics-module/confluent", newIsVer, newVer, newConstr, newVersion, version.StrategyRange, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected a change, got false")
	}
	if oldVersion != "invalid" {
		t.Errorf("expected old version 'invalid', got '%s'", oldVersion)
	}
	if resultVersion != newVersion {
		t.Errorf("expected new version '%s', got '%s'", newVersion, resultVersion)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the version was updated correctly
	expectedContent := `
module "kafka_topics_ziworkflows_module" {
  source  = "api.env0.com/kafka-topics-module/confluent"
  version = ">= 2.0.0, < 3.0.0"
}
`
	if string(updatedContent) != expectedContent {
		t.Errorf("Expected version replaced with %s. Got:\n%s", newVersion, string(updatedContent))
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

	_, _, _, err = UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, false, false)
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
  source = "test/test-module"
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

	_, _, _, err = UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, false, false)
	if err == nil {
		t.Error("Expected error for write-protected file, got nil")
	}
}

func TestUpdateModuleVersionInFile_DryRun(t *testing.T) {
	content := `
module "test_module" {
  source  = "test/test-module"
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

	changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(tfFile, "test-module", newIsVer, newVer, newConstr, "2.0.0", version.StrategyDynamic, true, false)
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
	data, err := os.ReadFile(tfFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != originalContent {
		t.Error("File was modified in dry-run mode")
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

func TestMatchModuleSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		pattern string
		want    bool
	}{
		{
			name:    "exact match",
			source:  "hashicorp/aws/vpc",
			pattern: "aws/vpc",
			want:    true,
		},
		{
			name:    "match at start",
			source:  "aws/vpc/module",
			pattern: "aws/vpc",
			want:    true,
		},
		{
			name:    "match in middle",
			source:  "registry.terraform.io/aws/vpc/latest",
			pattern: "aws/vpc",
			want:    true,
		},
		{
			name:    "no match - different segments",
			source:  "api.env0.com/test/foundations-service-account-module/google",
			pattern: "service-account-module",
			want:    false,
		},
		{
			name:    "no match - partial segment",
			source:  "hashicorp/aws-vpc/module",
			pattern: "aws",
			want:    false,
		},
		{
			name:    "single segment match",
			source:  "hashicorp/aws/vpc",
			pattern: "aws",
			want:    true,
		},
		{
			name:    "case sensitive",
			source:  "hashicorp/AWS/vpc",
			pattern: "aws",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchModuleSource(tt.source, tt.pattern)
			if got != tt.want {
				t.Errorf("matchModuleSource(%q, %q) = %v, want %v",
					tt.source, tt.pattern, got, tt.want)
			}
		})
	}
}
