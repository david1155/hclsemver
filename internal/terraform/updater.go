package terraform

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/david1155/hclsemver/pkg/version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ShouldProcessTier determines if a given path should be processed based on the config tiers
func ShouldProcessTier(path string, configTiers map[string]bool) bool {
	// If no tiers are configured, process all files
	if len(configTiers) == 0 {
		return true
	}

	// Extract potential tier from path
	parts := strings.Split(path, string(os.PathSeparator))
	for _, part := range parts {
		// Check if any configured tier is part of the path component
		for tier := range configTiers {
			// Check if tier is a directory name or part of the filename
			if part == tier || strings.Contains(part, tier) {
				return true
			}
		}
	}

	// If no tier matches were found in the path, process only if we're not in a tier-specific config
	return false
}

// ScanAndUpdateModules walks `rootDir`, searching for *.tf files.
// For each, calls UpdateModuleVersionInFile(...) to update module blocks if needed.
func ScanAndUpdateModules(
	workDir string,
	oldSourceSubstr string,
	newIsVer bool,
	newVer *semver.Version,
	newConstr *semver.Constraints,
	newInput string,
	configTiers map[string]bool,
	strategy version.Strategy,
) error {
	err := filepath.WalkDir(workDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".tf") {
			return nil
		}

		// Check if this file is in a tier we want to process
		if !ShouldProcessTier(path, configTiers) {
			return nil
		}

		changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(path, oldSourceSubstr, newIsVer, newVer, newConstr, newInput, strategy)
		if err != nil {
			return fmt.Errorf("error updating file %s: %w", path, err)
		}

		if changed {
			fmt.Printf("Updated file %s:\n", path)
			fmt.Printf("  - Version changed from '%s' to '%s'\n", oldVersion, newVersion)
			fmt.Printf("  - Strategy used: %s\n", strategy)
		}

		return nil
	})

	return err
}

// UpdateModuleVersionInFile reads a single .tf file, finds any module blocks
// whose "source" matches oldSourceSubstr, then updates "version" attribute using
// "keep old if it fits new, else new" logic. We pass newInput to decideVersionOrRange.
func UpdateModuleVersionInFile(
	filename string,
	oldSourceSubstr string,
	newIsVer bool,
	newVer *semver.Version,
	newConstr *semver.Constraints,
	newInput string,
	strategy version.Strategy,
) (bool, string, string, error) {
	// 1) Read file
	src, err := os.ReadFile(filename)
	if err != nil {
		return false, "", "", fmt.Errorf("cannot read file: %w", err)
	}

	// 2) Parse into AST
	file, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return false, "", "", fmt.Errorf("parse error in %s: %s", filename, diags.Error())
	}

	changed := false
	var oldVersion, newVersion string
	rootBody := file.Body()

	// Find module blocks
	for _, block := range rootBody.Blocks() {
		if block.Type() != "module" {
			continue
		}

		// Check if this is the module we want to update
		sourceAttr := block.Body().GetAttribute("source")
		if sourceAttr == nil {
			continue
		}

		sourceTokens := sourceAttr.Expr().BuildTokens(nil)
		sourceValue := strings.Trim(string(sourceTokens.Bytes()), `"`)

		if !strings.Contains(sourceValue, oldSourceSubstr) {
			continue
		}

		// Get existing version if any
		versionAttr := block.Body().GetAttribute("version")
		if versionAttr != nil {
			oldVersion = strings.Trim(strings.TrimSpace(string(versionAttr.Expr().BuildTokens(nil).Bytes())), `"`)
		}

		// Apply version strategy
		finalVersion, err := version.ApplyVersionStrategy(strategy, newInput, oldVersion)
		if err != nil {
			return false, "", "", fmt.Errorf("failed to apply version strategy: %w", err)
		}
		newVersion = finalVersion

		// Update the version attribute
		block.Body().SetAttributeValue("version", cty.StringVal(finalVersion))
		changed = true
	}

	if !changed {
		return false, "", "", nil
	}

	// Write the file back
	if err := os.WriteFile(filename, file.Bytes(), 0o644); err != nil {
		return false, "", "", fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return true, oldVersion, newVersion, nil
}
