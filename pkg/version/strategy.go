package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// isPre100Version checks if a version is below 1.0.0
func isPre100Version(v *semver.Version) bool {
	return v != nil && v.Major() == 0
}

// compareVersions compares two versions with special handling for pre-1.0 versions
func compareVersions(v1, v2 *semver.Version) int {
	if v1 == nil || v2 == nil {
		return 0
	}

	// If one is pre-1.0 and the other isn't, handle specially
	v1Pre := isPre100Version(v1)
	v2Pre := isPre100Version(v2)
	if v1Pre != v2Pre {
		// Pre-1.0 versions are considered lower than post-1.0 versions
		if v1Pre {
			return -1
		}
		return 1
	}

	// For pre-1.0 versions, compare normally but preserve metadata
	if v1Pre && v2Pre {
		if v1.Equal(v2) {
			return 0
		}
		if v1.GreaterThan(v2) {
			return 1
		}
		return -1
	}

	// For post-1.0 versions, use standard comparison
	if v1.Equal(v2) {
		return 0
	}
	if v1.GreaterThan(v2) {
		return 1
	}
	return -1
}

// DecideVersionOrRange does "keep old if it fits new, otherwise new."
func DecideVersionOrRange(
	oldIsVer bool,
	oldVer *semver.Version,
	oldRange *semver.Constraints,
	oldInput string,
	newIsVer bool,
	newVer *semver.Version,
	newRange *semver.Constraints,
	newInput string,
) string {
	switch {
	case oldIsVer && newIsVer:
		// Use enhanced version comparison
		comp := compareVersions(oldVer, newVer)
		if comp > 0 {
			return oldVer.Original()
		}
		return newVer.Original()

	case oldIsVer && !newIsVer:
		// If old version is exact and new is a range, first check if old version is higher than any version in the range
		maxVer := findHighestVersionInRange(newRange)
		if maxVer != nil && compareVersions(oldVer, maxVer) > 0 {
			return oldVer.Original()
		}
		// If old version fits in the new range, keep old version for consistency
		if newRange.Check(oldVer) {
			return oldVer.Original()
		}
		// For backward protection, if old version is higher than the minimum of the new range,
		// keep the old version
		minVer := findLowestVersionInRange(newRange)
		if minVer != nil && compareVersions(oldVer, minVer) > 0 {
			return oldVer.Original()
		}
		// Otherwise use the new range
		return newInput

	case !oldIsVer && newIsVer:
		if oldRange != nil {
			// If old range has a higher minimum version, keep old range
			minVer := findLowestVersionInRange(oldRange)
			if minVer != nil && compareVersions(minVer, newVer) > 0 {
				return oldInput
			}
			// If old range has a higher maximum version, keep old range
			maxVer := findHighestVersionInRange(oldRange)
			if maxVer != nil && compareVersions(maxVer, newVer) > 0 {
				return oldInput
			}
			// If new version fits in old range, keep old range for consistency
			if oldRange.Check(newVer) {
				return oldInput
			}
		}
		// Use new exact version
		return newVer.Original()

	default:
		// Both are ranges
		if oldRange == nil || newRange == nil {
			return newInput
		}

		// Find highest and lowest versions in both ranges
		oldMaxVer := findHighestVersionInRange(oldRange)
		newMaxVer := findHighestVersionInRange(newRange)
		oldMinVer := findLowestVersionInRange(oldRange)
		newMinVer := findLowestVersionInRange(newRange)

		// If old range has higher minimum version than new range, keep old range
		if oldMinVer != nil && newMinVer != nil && compareVersions(oldMinVer, newMinVer) > 0 {
			return oldInput
		}

		// If old range has higher version than new range, keep old range
		if oldMaxVer != nil && newMaxVer != nil && compareVersions(oldMaxVer, newMaxVer) > 0 {
			return oldInput
		}

		// If ranges overlap, keep old range for consistency
		if RangesOverlap(oldRange, newRange) {
			return oldInput
		}

		return newInput
	}
}

// ApplyVersionStrategy applies the specified strategy to convert between version formats
func ApplyVersionStrategy(strategy Strategy, targetVersion string, existingVersion string) (string, error) {
	switch strategy {
	case StrategyExact:
		// First, parse both versions
		targetVer, err := semver.NewVersion(targetVersion)
		if err != nil {
			return "", fmt.Errorf("exact strategy requires an exact version (e.g., '2.1.1'), got: %s", targetVersion)
		}

		// If no existing version, use target version
		if existingVersion == "" {
			return targetVer.String(), nil
		}

		// Parse existing version
		existingVer, err := semver.NewVersion(existingVersion)
		if err != nil {
			// If existing version is invalid, use target version
			return targetVer.String(), nil
		}

		// For backward compatibility protection, if existing version is higher, keep it
		if existingVer.GreaterThan(targetVer) {
			return existingVer.String(), nil
		}

		return targetVer.String(), nil

	case StrategyRange:
		return ApplyRangeStrategy(targetVersion, existingVersion)
	case StrategyDynamic:
		return ApplyDynamicStrategy(targetVersion, existingVersion)
	default:
		return targetVersion, nil
	}
}

func ConvertToExactVersion(version string) (string, error) {
	// For exact strategy, only accept exact versions
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("exact strategy requires an exact version (e.g., '2.1.1'), got: %s", version)
	}
	return v.String(), nil
}

// preserveVersionMetadata returns the original version string if it's a pre-1.0 version,
// preserving any pre-release tags and build metadata
func preserveVersionMetadata(v *semver.Version) string {
	if v == nil {
		return ""
	}
	if isPre100Version(v) {
		return v.Original()
	}
	return v.String()
}

// splitRangeAtVersion splits a range at a specific version, returning the parts before and after
func splitRangeAtVersion(r *semver.Constraints, v *semver.Version) (*semver.Constraints, *semver.Constraints) {
	if r == nil || v == nil {
		return nil, nil
	}

	// Get the string representation of the range
	rangeStr := normalizeVersionString(r.String())

	// Handle OR conditions
	if strings.Contains(rangeStr, "||") {
		parts := strings.Split(rangeStr, "||")
		var beforeParts, afterParts []string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			c, err := semver.NewConstraint(part)
			if err != nil {
				continue
			}

			minVer := findLowestVersionInRange(c)
			if minVer != nil && minVer.LessThan(v) {
				beforeParts = append(beforeParts, part)
			} else {
				afterParts = append(afterParts, part)
			}
		}

		var before, after *semver.Constraints
		if len(beforeParts) > 0 {
			before, _ = semver.NewConstraint(strings.Join(beforeParts, " || "))
		}
		if len(afterParts) > 0 {
			after, _ = semver.NewConstraint(strings.Join(afterParts, " || "))
		}
		return before, after
	}

	// Handle single range
	minVer := findLowestVersionInRange(r)
	maxVer := findHighestVersionInRange(r)

	if minVer == nil || maxVer == nil {
		return nil, nil
	}

	if minVer.GreaterThan(v) {
		return nil, r
	}
	if maxVer.LessThan(v) {
		return r, nil
	}

	// Create the split ranges
	beforeStr := fmt.Sprintf(">=%s,<%s", minVer.String(), v.String())
	afterStr := fmt.Sprintf(">=%s,<%s", v.String(), maxVer.String())
	before, _ := semver.NewConstraint(beforeStr)
	after, _ := semver.NewConstraint(afterStr)

	return before, after
}

// handleComplexRange processes complex version ranges, handling OR conditions and splits at 1.0.0
func handleComplexRange(version string) (string, error) {
	// Split by OR conditions first
	parts := strings.Split(version, "||")
	for _, part := range parts {
		// Parse each part as a constraint
		c, err := semver.NewConstraint(strings.TrimSpace(part))
		if err != nil {
			continue
		}

		// Get the minimum version from this constraint
		v, err := getMinVersionFromConstraint(c)
		if err != nil {
			continue
		}

		// If this is a pre-1.0 version, return it with its metadata
		if isPre100Version(v) {
			// Try to extract the exact version with metadata from the original string
			rangeParts := strings.Split(strings.TrimSpace(part), ",")
			for _, rangePart := range rangeParts {
				rangePart = strings.TrimSpace(rangePart)
				if strings.Contains(rangePart, v.String()) {
					// Extract the exact version with metadata
					if exactV, err := semver.NewVersion(strings.TrimLeft(rangePart, ">=<")); err == nil {
						return exactV.Original(), nil
					}
				}
			}
			return v.Original(), nil
		}
	}

	// If no pre-1.0 version found, check if it's a post-1.0 range
	c, err := semver.NewConstraint(version)
	if err != nil {
		return "", err
	}

	v, err := getMinVersionFromConstraint(c)
	if err != nil {
		return "", err
	}

	// For post-1.0 versions, preserve the range format
	if !isPre100Version(v) {
		return normalizeVersionString(version), nil
	}

	return v.Original(), nil
}

func ConvertToRangeVersion(version string) (string, error) {
	// Handle complex ranges first
	if strings.Contains(version, "||") || strings.Contains(version, "~>") {
		return handleComplexRange(version)
	}

	// If it's already a range, normalize and return as is
	if _, err := semver.NewConstraint(version); err == nil && strings.Contains(version, ">") {
		// For pre-1.0 ranges, convert to exact version using the minimum
		c, _ := semver.NewConstraint(version)
		minVer := findLowestVersionInRange(c)
		if isPre100Version(minVer) {
			// Extract the exact version with metadata from the original string
			parts := strings.Split(version, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.Contains(part, minVer.String()) {
					// Extract the exact version with metadata
					if v, err := semver.NewVersion(strings.TrimLeft(part, ">=<")); err == nil {
						return v.Original(), nil
					}
				}
			}
			return minVer.Original(), nil
		}
		// Normalize the version string
		return normalizeVersionString(version), nil
	}

	// Parse as exact version
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// Keep pre-1.0 versions as exact with metadata preserved
	if isPre100Version(v) {
		return v.Original(), nil
	}

	// Convert to range >=current,<next-major with consistent spacing
	return normalizeVersionString(fmt.Sprintf(">=%s,<%d.0.0", v.String(), v.Major()+1)), nil
}

func ApplyRangeStrategy(targetVersion, existingVersion string) (string, error) {
	// If no existing version, convert target to range
	if existingVersion == "" {
		// Expand tilde arrow notation first
		expandedTarget := ExpandTerraformTildeArrow(targetVersion)
		return ConvertToRangeVersion(expandedTarget)
	}

	// Expand tilde arrow notation
	expandedTarget := ExpandTerraformTildeArrow(targetVersion)
	expandedExisting := ExpandTerraformTildeArrow(existingVersion)

	// Parse target version
	targetIsVer, targetVer, targetRange, err := ParseVersionOrRange(expandedTarget)
	if err != nil {
		return "", fmt.Errorf("invalid target version: %w", err)
	}

	// Parse existing version
	existingIsVer, existingVer, existingRange, err := ParseVersionOrRange(expandedExisting)
	if err != nil {
		// If existing version is invalid, convert target to range
		return ConvertToRangeVersion(expandedTarget)
	}

	// Handle pre-1.0 versions
	if targetIsVer && isPre100Version(targetVer) {
		// If existing version is higher, keep it
		if existingIsVer && existingVer != nil && existingVer.GreaterThan(targetVer) {
			return preserveVersionMetadata(existingVer), nil
		}
		return preserveVersionMetadata(targetVer), nil
	}

	// Handle pre-1.0 target range
	if !targetIsVer && targetRange != nil {
		minVer := findLowestVersionInRange(targetRange)
		if isPre100Version(minVer) {
			// If existing version is higher, keep it
			if existingIsVer && existingVer != nil && existingVer.GreaterThan(minVer) {
				return preserveVersionMetadata(existingVer), nil
			}
			return preserveVersionMetadata(minVer), nil
		}
	}

	// If existing is pre-1.0 exact version and higher than target, keep it
	if existingIsVer && existingVer != nil && isPre100Version(existingVer) {
		if targetIsVer && targetVer != nil && existingVer.GreaterThan(targetVer) {
			return preserveVersionMetadata(existingVer), nil
		}
		if !targetIsVer && targetRange != nil {
			minVer := findLowestVersionInRange(targetRange)
			if minVer != nil && existingVer.GreaterThan(minVer) {
				return preserveVersionMetadata(existingVer), nil
			}
		}
	}

	// If existing is a range and target is a version that fits in it, keep existing range
	if !existingIsVer && existingRange != nil && targetIsVer && targetVer != nil {
		if existingRange.Check(targetVer) {
			return normalizeVersionString(expandedExisting), nil
		}
	}

	// If existing is a range with higher minimum version, keep it
	if !existingIsVer && existingRange != nil && targetIsVer && targetVer != nil {
		existingMinVer := findLowestVersionInRange(existingRange)
		if existingMinVer != nil && existingMinVer.GreaterThan(targetVer) {
			return normalizeVersionString(expandedExisting), nil
		}
	}

	// If target is already a range, normalize and return it
	if !targetIsVer && targetRange != nil {
		return normalizeVersionString(expandedTarget), nil
	}

	// Otherwise convert target to range
	return ConvertToRangeVersion(expandedTarget)
}

func ApplyDynamicStrategy(targetVersion, existingVersion string) (string, error) {
	// If no existing version, use target as is
	if existingVersion == "" {
		// For pre-1.0 ranges, convert to exact version
		if _, err := semver.NewConstraint(targetVersion); err == nil && strings.Contains(targetVersion, ">") {
			c, _ := semver.NewConstraint(targetVersion)
			minVer := findLowestVersionInRange(c)
			if isPre100Version(minVer) {
				return preserveVersionMetadata(minVer), nil
			}
		}
		return targetVersion, nil
	}

	// Expand tilde arrow notation first
	expandedTarget := ExpandTerraformTildeArrow(targetVersion)
	expandedExisting := ExpandTerraformTildeArrow(existingVersion)

	// Parse target version/range
	targetIsVer, targetVer, targetRange, err := ParseVersionOrRange(expandedTarget)
	if err != nil {
		return "", fmt.Errorf("invalid target version: %w", err)
	}

	// Parse existing version/range
	existingIsVer, existingVer, existingRange, err := ParseVersionOrRange(expandedExisting)
	if err != nil {
		// If existing version is invalid, use target as is
		return targetVersion, nil
	}

	// Handle pre-1.0 target version
	if targetIsVer && isPre100Version(targetVer) {
		// If existing version is higher, keep it
		if existingIsVer && existingVer != nil && existingVer.GreaterThan(targetVer) {
			return preserveVersionMetadata(existingVer), nil
		}
		// If existing is a range and its minimum version is higher, keep it
		if !existingIsVer && existingRange != nil {
			minVer := findLowestVersionInRange(existingRange)
			if minVer != nil && isPre100Version(minVer) && minVer.GreaterThan(targetVer) {
				return normalizeVersionString(expandedExisting), nil
			}
		}
		return preserveVersionMetadata(targetVer), nil
	}

	// Handle pre-1.0 target range
	if !targetIsVer && targetRange != nil {
		targetMinVer := findLowestVersionInRange(targetRange)
		if isPre100Version(targetMinVer) {
			// If existing is a range, check if it's higher
			if !existingIsVer && existingRange != nil {
				existingMinVer := findLowestVersionInRange(existingRange)
				if existingMinVer != nil && existingMinVer.GreaterThan(targetMinVer) {
					// If both are pre-1.0 ranges, keep the existing range
					if isPre100Version(existingMinVer) {
						return normalizeVersionString(expandedExisting), nil
					}
				}
			}
			// If existing is exact and higher, keep it
			if existingIsVer && existingVer != nil && existingVer.GreaterThan(targetMinVer) {
				return preserveVersionMetadata(existingVer), nil
			}
			// Convert pre-1.0 range to exact version
			return preserveVersionMetadata(targetMinVer), nil
		}
	}

	// If existing is pre-1.0 exact version and higher than target, keep it
	if existingIsVer && existingVer != nil && isPre100Version(existingVer) {
		if targetIsVer && targetVer != nil && existingVer.GreaterThan(targetVer) {
			return preserveVersionMetadata(existingVer), nil
		}
		if !targetIsVer && targetRange != nil {
			minVer := findLowestVersionInRange(targetRange)
			if minVer != nil && existingVer.GreaterThan(minVer) {
				return preserveVersionMetadata(existingVer), nil
			}
		}
	}

	// If existing is a pre-1.0 range
	if !existingIsVer && existingRange != nil {
		existingMinVer := findLowestVersionInRange(existingRange)
		if isPre100Version(existingMinVer) {
			// If target is post-1.0, keep existing range
			if targetIsVer && targetVer != nil && targetVer.Major() > 0 {
				return normalizeVersionString(expandedExisting), nil
			}
			if !targetIsVer && targetRange != nil {
				targetMinVer := findLowestVersionInRange(targetRange)
				if targetMinVer != nil && targetMinVer.Major() > 0 {
					return normalizeVersionString(expandedExisting), nil
				}
			}
		}
	}

	// If existing is a range and target is an exact version outside the range,
	// convert target to a range with the same format
	if !existingIsVer && existingRange != nil && targetIsVer && !existingRange.Check(targetVer) {
		nextMajor := targetVer.Major() + 1
		expandedTarget = fmt.Sprintf(">=%d, <%d", targetVer.Major(), nextMajor)
		targetIsVer = false
		targetRange, _ = semver.NewConstraint(expandedTarget)
	}

	// Use DecideVersionOrRange to handle all cases consistently
	result := DecideVersionOrRange(
		existingIsVer, existingVer, existingRange, expandedExisting,
		targetIsVer, targetVer, targetRange, expandedTarget,
	)

	// Normalize the result
	return normalizeVersionString(result), nil
}

// getMinVersionFromConstraint extracts the minimum version from a constraint
func getMinVersionFromConstraint(c *semver.Constraints) (*semver.Version, error) {
	// Start with a very low version to find the minimum that satisfies the constraint
	v, err := semver.NewVersion("0.0.0")
	if err != nil {
		return nil, err
	}

	// Keep incrementing until we find a version that satisfies the constraint
	for !c.Check(v) {
		if v.Major() == 0 {
			if v.Minor() == 99 {
				v, _ = semver.NewVersion("1.0.0")
			} else {
				v, _ = semver.NewVersion(fmt.Sprintf("0.%d.0", v.Minor()+1))
			}
		} else {
			break // Don't go beyond major version 1
		}
	}

	// Extract the exact version with metadata from the original constraint string
	parts := strings.Split(c.String(), ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, v.String()) {
			// Try to extract version with metadata
			if exactV, err := semver.NewVersion(strings.TrimLeft(part, ">=<")); err == nil {
				return exactV, nil
			}
		}
	}

	return v, nil
}
