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

	// First check for specific tier matches
	for _, part := range parts {
		for tier := range configTiers {
			if tier == "*" {
				continue
			}
			// Check if tier is a directory name or part of the filename
			if part == tier || strings.Contains(part, tier) {
				return configTiers[tier] // Return the specific tier's setting
			}
		}
	}

	// If we have only "*" configured, use its value
	if len(configTiers) == 1 && configTiers["*"] {
		return true
	}

	// If we have specific tiers and "*", and no specific tier matched,
	// use "*" as the default
	if wildcardValue, hasWildcard := configTiers["*"]; hasWildcard {
		return wildcardValue
	}

	// If no tier matches were found and no wildcard, don't process
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
	dryRun bool,
	force bool,
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

		changed, oldVersion, newVersion, err := UpdateModuleVersionInFile(path, oldSourceSubstr, newIsVer, newVer, newConstr, newInput, strategy, dryRun, force)
		if err != nil {
			return fmt.Errorf("error updating file %s: %w", path, err)
		}

		if changed {
			if dryRun {
				fmt.Printf("[DRY RUN] Would update file %s:\n", path)
				fmt.Printf("  - Would change version from '%s' to '%s'\n", oldVersion, newVersion)
				fmt.Printf("  - Strategy that would be used: %s\n", strategy)
			} else {
				fmt.Printf("Updated file %s:\n", path)
				fmt.Printf("  - Version changed from '%s' to '%s'\n", oldVersion, newVersion)
				fmt.Printf("  - Strategy used: %s\n", strategy)
			}
		}

		return nil
	})

	return err
}

// matchModuleSource checks if the source matches the pattern by comparing path segments
func matchModuleSource(source, pattern string) bool {
	// Split both strings by forward slash
	sourceParts := strings.Split(source, "/")
	patternParts := strings.Split(pattern, "/")

	// For each part in the source
	for i := 0; i <= len(sourceParts)-len(patternParts); i++ {
		matched := true
		// Try to match pattern parts with source parts starting at current position
		for j := 0; j < len(patternParts); j++ {
			if sourceParts[i+j] != patternParts[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
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
	dryRun bool,
	force bool,
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

		if !matchModuleSource(sourceValue, oldSourceSubstr) {
			continue
		}

		// Get existing version if any
		versionAttr := block.Body().GetAttribute("version")
		if versionAttr != nil {
			oldVersion = strings.Trim(strings.TrimSpace(string(versionAttr.Expr().BuildTokens(nil).Bytes())), `"`)
		} else if !force {
			// If no version attribute and force is false, output warning and skip
			fmt.Printf("Warning: Module %q in file %s has no version attribute. Use force flag to add version.\n", sourceValue, filename)
			continue
		}

		// Apply version strategy
		finalVersion, err := version.ApplyVersionStrategy(strategy, newInput, oldVersion)
		if err != nil {
			return false, "", "", fmt.Errorf("failed to apply version strategy: %w", err)
		}
		newVersion = finalVersion

		// Normalize both versions for comparison
		normalizedOld := version.NormalizeVersionString(oldVersion)
		normalizedNew := version.NormalizeVersionString(finalVersion)

		// Only update if the normalized versions are different
		if normalizedOld != normalizedNew {
			// Update the version attribute
			block.Body().SetAttributeValue("version", cty.StringVal(finalVersion))
			changed = true
		}
	}

	if !changed {
		return false, oldVersion, "", nil
	}

	if !dryRun {
		// Write the file back
		if err := os.WriteFile(filename, file.Bytes(), 0o644); err != nil {
			return false, "", "", fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return true, oldVersion, newVersion, nil
}
