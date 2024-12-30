package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

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
		// If old version is greater than new version, keep old version
		if oldVer.GreaterThan(newVer) {
			return oldVer.Original()
		}
		return newVer.Original()

	case oldIsVer && !newIsVer:
		// If old version is exact and new is a range, first check if old version is higher than any version in the range
		maxVer := findHighestVersionInRange(newRange)
		if maxVer != nil && oldVer.GreaterThan(maxVer) {
			return oldVer.Original()
		}
		// If old version fits in the new range, keep old version for consistency
		if newRange.Check(oldVer) {
			return oldVer.Original()
		}
		// For backward protection, if old version is higher than the minimum of the new range,
		// keep the old version
		minVer := findLowestVersionInRange(newRange)
		if minVer != nil && oldVer.GreaterThan(minVer) {
			return oldVer.Original()
		}
		// Otherwise use the new range
		return newInput

	case !oldIsVer && newIsVer:
		if oldRange != nil {
			// If old range has a higher minimum version, keep old range
			minVer := findLowestVersionInRange(oldRange)
			if minVer != nil && minVer.GreaterThan(newVer) {
				return oldInput
			}
			// If old range has a higher maximum version, keep old range
			maxVer := findHighestVersionInRange(oldRange)
			if maxVer != nil && maxVer.GreaterThan(newVer) {
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
		if oldMinVer != nil && newMinVer != nil && oldMinVer.GreaterThan(newMinVer) {
			return oldInput
		}

		// If old range has higher version than new range, keep old range
		if oldMaxVer != nil && newMaxVer != nil && oldMaxVer.GreaterThan(newMaxVer) {
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

func ConvertToRangeVersion(version string) (string, error) {
	// If it's already a range, normalize and return as is
	if _, err := semver.NewConstraint(version); err == nil && strings.Contains(version, ">") {
		// Normalize the version string
		return normalizeVersionString(version), nil
	}

	// Parse as exact version
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
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
	existingIsVer, _, existingRange, err := ParseVersionOrRange(expandedExisting)
	if err != nil {
		// If existing version is invalid, convert target to range
		return ConvertToRangeVersion(expandedTarget)
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
