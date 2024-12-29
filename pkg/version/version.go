package version

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	MAX_MAJOR = 20
	MAX_MINOR = 50
	MAX_PATCH = 50
)

type Strategy string

const (
	StrategyDynamic Strategy = "dynamic"
	StrategyExact   Strategy = "exact"
	StrategyRange   Strategy = "range"
)

// ParseVersionOrRange tries single version (e.g. "1.2.3") first; if that fails,
// tries a range with "~>" expansions.
func ParseVersionOrRange(input string) (bool, *semver.Version, *semver.Constraints, error) {
	if input == "" {
		return false, nil, nil, fmt.Errorf("empty version input")
	}

	v, errVer := semver.NewVersion(input)
	if errVer == nil {
		return true, v, nil, nil
	}

	tfInput := ExpandTerraformTildeArrow(input)
	c, errConstr := semver.NewConstraint(tfInput)
	if errConstr == nil {
		return false, nil, c, nil
	}
	return false, nil, nil, errConstr
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
		// If old version is greater than new version, keep old version
		if oldVer.GreaterThan(newVer) {
			return oldVer.Original()
		}
		return newVer.Original()

	case oldIsVer && !newIsVer:
		// If old version is exact and new is a range, check if old version is greater than any version in the range
		if newRange.Check(oldVer) {
			return oldVer.Original()
		}
		// Try to find the highest version in the new range
		maxVer := findHighestVersionInRange(newRange)
		if maxVer != nil && oldVer.GreaterThan(maxVer) {
			return oldVer.Original()
		}
		return newInput

	case !oldIsVer && newIsVer:
		if oldRange != nil && oldRange.Check(newVer) {
			return oldInput
		}
		// Try to find the highest version in the old range
		maxVer := findHighestVersionInRange(oldRange)
		if maxVer != nil && maxVer.GreaterThan(newVer) {
			return oldInput
		}
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

// findHighestVersionInRange tries to find the highest version that satisfies the constraints
// by checking common version patterns
func findHighestVersionInRange(c *semver.Constraints) *semver.Version {
	if c == nil {
		return nil
	}

	// Try to find the highest version that satisfies the constraints
	var highestVer *semver.Version
	for major := 0; major <= MAX_MAJOR; major++ {
		for minor := 0; minor <= MAX_MINOR; minor++ {
			for patch := 0; patch <= MAX_PATCH; patch++ {
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
				if c.Check(testVer) {
					if highestVer == nil || testVer.GreaterThan(highestVer) {
						highestVer = testVer
					}
				}
			}
		}
	}
	return highestVer
}

// findLowestVersionInRange tries to find the lowest version that satisfies the constraints
func findLowestVersionInRange(c *semver.Constraints) *semver.Version {
	if c == nil {
		return nil
	}

	var lowestVer *semver.Version
	for major := 0; major <= MAX_MAJOR; major++ {
		for minor := 0; minor <= MAX_MINOR; minor++ {
			for patch := 0; patch <= MAX_PATCH; patch++ {
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
				if c.Check(testVer) {
					if lowestVer == nil || testVer.LessThan(lowestVer) {
						lowestVer = testVer
					}
				}
			}
		}
	}
	return lowestVer
}

// RangesOverlap tries some sample versions from 0.0.0..(MAX_MAJOR,MAX_MINOR,MAX_PATCH).
func RangesOverlap(a, b *semver.Constraints) bool {
	for major := 0; major <= MAX_MAJOR; major++ {
		for minor := 0; minor <= MAX_MINOR; minor++ {
			for patch := 0; patch <= MAX_PATCH; patch++ {
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
				if a.Check(testVer) && b.Check(testVer) {
					return true
				}
			}
		}
	}
	return false
}

// ExpandTerraformTildeArrow scans for "~>" => ">=X.Y.Z,<X+1.0.0"
func ExpandTerraformTildeArrow(version string) string {
	if version == "" {
		return version
	}

	// If it's not a tilde arrow version, return as is
	if !strings.Contains(version, "~>") {
		return version
	}

	var result []string
	for _, part := range strings.Split(version, "||") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "~>") {
			part = strings.TrimPrefix(part, "~>")
			part = strings.TrimSpace(part)
			result = append(result, buildRangeFromTildePart(part))
		} else {
			result = append(result, part)
		}
	}

	return strings.Join(result, " || ")
}

func readToken(s string) (token, remainder string) {
	seps := []int{}
	for _, sep := range []string{" ", ",", "||"} {
		if i := strings.Index(s, sep); i != -1 {
			seps = append(seps, i)
		}
	}
	if len(seps) == 0 {
		return s, ""
	}
	min := seps[0]
	for _, v := range seps {
		if v < min {
			min = v
		}
	}
	return s[:min], s[min:]
}

func buildRangeFromTildePart(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "~>MISSING"
	}

	parts := strings.Split(version, ".")
	if len(parts) > 3 {
		return "~>INVALID"
	}

	// Parse the version
	ver, err := semver.NewVersion(version)
	if err != nil {
		return ">=0.0.0, <1.0.0"
	}

	// Calculate the next major version
	nextMajor := ver.Major() + 1

	// Return the range without spaces after operators
	return fmt.Sprintf(">=%d.%d.%d, <%d.0.0", ver.Major(), ver.Minor(), ver.Patch(), nextMajor)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// ApplyVersionStrategy applies the specified strategy to convert between version formats
func ApplyVersionStrategy(strategy Strategy, targetVersion string, existingVersion string) (string, error) {
	switch strategy {
	case StrategyExact:
		return ConvertToExactVersion(targetVersion)
	case StrategyRange:
		return ConvertToRangeVersion(targetVersion)
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
	// If it's already a range, return as is
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

// normalizeVersionString ensures consistent formatting of version strings
func normalizeVersionString(version string) string {
	// Remove all spaces first
	version = strings.ReplaceAll(version, " ", "")

	// Add spaces after operators and commas
	version = strings.ReplaceAll(version, ">=", ">= ")
	version = strings.ReplaceAll(version, "<", "< ")
	version = strings.ReplaceAll(version, ",", ", ")

	// Remove any trailing spaces
	version = strings.TrimSpace(version)

	return version
}
