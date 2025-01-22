package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/david1155/hclsemver/pkg/version"
)

func TestUpdateModuleVersionInFile(t *testing.T) {
	// Create a temporary directory for test files
	dir, err := os.MkdirTemp("", "TestUpdateModuleVersionInFile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Test cases
	tests := []struct {
		name        string
		content     string
		newVersion  string
		wantChanged bool
		wantOld     string
		wantNew     string
	}{
		{
			name: "no space after comma in range",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">=1.0.0,<2.0.0"
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">=1.0.0,<2.0.0",
			wantNew:     "",
		},
		{
			name: "inconsistent spaces in range",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">=1.0.0, <2.0.0"
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">=1.0.0, <2.0.0",
			wantNew:     "",
		},
		{
			name: "extra spaces in range",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">=  1.0.0,  <  2.0.0"
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">=  1.0.0,  <  2.0.0",
			wantNew:     "",
		},
		{
			name: "no spaces at all in range",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">=1.0.0<2.0.0"
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">=1.0.0<2.0.0",
			wantNew:     "",
		},
		{
			name: "spaces in version numbers",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">= 1.0.0 , < 2.0.0 "
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">= 1.0.0 , < 2.0.0 ",
			wantNew:     "",
		},
		{
			name: "mixed spaces in operators",
			content: `
module "test_module" {
  source  = "api.env0.com/test-module/test"
  version = ">=1.0.0, <2.0.0"
}`,
			newVersion:  ">= 1.0.0, < 2.0.0",
			wantChanged: false,
			wantOld:     ">=1.0.0, <2.0.0",
			wantNew:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(dir, "test.tf")
			err = os.WriteFile(testFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Parse new version
			newIsVer, newVer, newConstr, err := version.ParseVersionOrRange(tc.newVersion)
			if err != nil {
				t.Fatal(err)
			}

			// Test updating the version
			changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(testFile, "test-module", newIsVer, newVer, newConstr, tc.newVersion, version.StrategyRange, false, false)
			if err != nil {
				t.Fatal(err)
			}

			if changed != tc.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if oldVersion != tc.wantOld {
				t.Errorf("oldVersion = %q, want %q", oldVersion, tc.wantOld)
			}
			if newVersion != tc.wantNew {
				t.Errorf("newVersion = %q, want %q", newVersion, tc.wantNew)
			}
		})
	}

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
		{
			name: "wildcard tier only",
			path: "/work/any/path/file.tf",
			configTiers: map[string]bool{
				"*": true,
			},
			want: true,
		},
		{
			name: "wildcard tier with specific tier - specific tier path",
			path: "/work/dev/module/file.tf",
			configTiers: map[string]bool{
				"*":   true,
				"dev": false,
			},
			want: false, // Specific tier setting takes precedence
		},
		{
			name: "wildcard tier with specific tier - other path",
			path: "/work/other/module/file.tf",
			configTiers: map[string]bool{
				"*":   true,
				"dev": false,
			},
			want: true, // Uses wildcard for non-matching paths
		},
		{
			name: "wildcard tier should not match as string",
			path: "/work/*/module/file.tf",
			configTiers: map[string]bool{
				"dev": true,
				"prd": true,
			},
			want: false,
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

func TestScanAndUpdateModules_Tiers(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create test directory structure
	dirs := []string{"dev", "stg", "prd", "other", "random/nested/path", "some/other/location"}
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		"dev/main.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
		"stg/main.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
		"prd/main.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
		"other/main.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
		"random/nested/path/resources.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
		"some/other/location/terraform.tf": `
module "test" {
  source  = "hashicorp/test-module/aws"
  version = "1.0.0"
}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test cases
	tests := []struct {
		name        string
		configTiers map[string]bool
		wantChanged map[string]bool
	}{
		{
			name: "specific tiers only",
			configTiers: map[string]bool{
				"dev": true,
				"stg": true,
				"prd": true,
			},
			wantChanged: map[string]bool{
				"dev/main.tf":   true,
				"stg/main.tf":   true,
				"prd/main.tf":   true,
				"other/main.tf": false,
			},
		},
		{
			name: "dev tier only",
			configTiers: map[string]bool{
				"dev": true,
			},
			wantChanged: map[string]bool{
				"dev/main.tf":   true,
				"stg/main.tf":   false,
				"prd/main.tf":   false,
				"other/main.tf": false,
			},
		},
		{
			name: "wildcard tier",
			configTiers: map[string]bool{
				"*": true,
			},
			wantChanged: map[string]bool{
				"dev/main.tf":                      true,
				"stg/main.tf":                      true,
				"prd/main.tf":                      true,
				"other/main.tf":                    true,
				"random/nested/path/resources.tf":  true,
				"some/other/location/terraform.tf": true,
			},
		},
		{
			name: "wildcard as default with different version for dev",
			configTiers: map[string]bool{
				"*":   true,  // Default for all tiers
				"dev": false, // Dev tier should not be processed
			},
			wantChanged: map[string]bool{
				"dev/main.tf":   false, // Should not change due to specific tier setting
				"stg/main.tf":   true,  // Should change due to wildcard
				"prd/main.tf":   true,  // Should change due to wildcard
				"other/main.tf": true,  // Should change due to wildcard
			},
		},
		{
			name:        "empty tiers (should process all)",
			configTiers: map[string]bool{},
			wantChanged: map[string]bool{
				"dev/main.tf":   true,
				"stg/main.tf":   true,
				"prd/main.tf":   true,
				"other/main.tf": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First, ensure all files have original content
			for filePath, content := range testFiles {
				fullPath := filepath.Join(tmpDir, filePath)
				err := os.WriteFile(fullPath, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to reset file: %v", err)
				}
			}

			if tt.name == "wildcard as default with different version for dev" {
				// Call ScanAndUpdateModules once with both wildcard and specific tier
				err := ScanAndUpdateModules(
					tmpDir,
					"test-module/aws",
					true,
					semver.MustParse("2.0.0"),
					nil,
					"2.0.0",
					tt.configTiers,
					version.StrategyExact,
					false,
					false,
				)
				if err != nil {
					t.Fatalf("ScanAndUpdateModules failed: %v", err)
				}

				// Verify the versions
				for filePath, shouldChange := range tt.wantChanged {
					fullPath := filepath.Join(tmpDir, filePath)
					content, err := os.ReadFile(fullPath)
					if err != nil {
						t.Fatalf("Failed to read file: %v", err)
					}

					if shouldChange {
						if !strings.Contains(string(content), `version = "2.0.0"`) {
							t.Errorf("File %s: expected version 2.0.0", filePath)
						}
					} else {
						if !strings.Contains(string(content), `version = "1.0.0"`) {
							t.Errorf("File %s: expected version 1.0.0", filePath)
						}
					}
				}
				return
			}

			// Call ScanAndUpdateModules once for other test cases
			err := ScanAndUpdateModules(
				tmpDir,
				"test-module/aws",
				true,
				semver.MustParse("2.0.0"),
				nil,
				"2.0.0",
				tt.configTiers,
				version.StrategyExact,
				false,
				false,
			)
			if err != nil {
				t.Fatalf("ScanAndUpdateModules failed: %v", err)
			}

			// Then check all files
			for filePath, shouldChange := range tt.wantChanged {
				fullPath := filepath.Join(tmpDir, filePath)
				updatedContent, err := os.ReadFile(fullPath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}

				wasChanged := string(updatedContent) != testFiles[filePath]
				if wasChanged != shouldChange {
					t.Errorf("File %s: expected changed=%v, got changed=%v", filePath, shouldChange, wasChanged)
				}
			}
		})
	}
}
