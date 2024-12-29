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

		// Find highest versions in both ranges
		oldMaxVer := findHighestVersionInRange(oldRange)
		newMaxVer := findHighestVersionInRange(newRange)

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
func ExpandTerraformTildeArrow(input string) string {
	out := input
	for {
		idx := strings.Index(out, "~>")
		if idx == -1 {
			break
		}
		rest := out[idx+2:]
		token, remainder := readToken(rest)
		rangePart := buildRangeFromTildePart(strings.TrimSpace(token))
		before := out[:idx]
		out = before + rangePart + remainder
	}
	return out
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

func buildRangeFromTildePart(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "~>MISSING"
	}
	parts := strings.Split(raw, ".")
	switch len(parts) {
	case 1:
		major := atoi(parts[0])
		return fmt.Sprintf(">=%d.0.0, <%d.0.0", major, major+1)
	case 2:
		major := atoi(parts[0])
		return fmt.Sprintf(">=%d.%s.0, <%d.0.0", major, parts[1], major+1)
	case 3:
		major := atoi(parts[0])
		minor := parts[1]
		patch := parts[2]
		return fmt.Sprintf(">=%d.%s.%s, <%d.0.0", major, minor, patch, major+1)
	default:
		return "~>INVALID"
	}
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
		// Ensure consistent spacing
		version = strings.ReplaceAll(version, ",", ", ")
		version = strings.ReplaceAll(version, "  ", " ")
		return version, nil
	}

	// Parse as exact version
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	// Convert to range >=current,<next-major with consistent spacing
	return fmt.Sprintf(">=%s, <%d.0.0", v.String(), v.Major()+1), nil
}

func ApplyDynamicStrategy(targetVersion, existingVersion string) (string, error) {
	// If no existing version, use target as is
	if existingVersion == "" {
		return targetVersion, nil
	}

	// Parse target version/range
	targetIsVer, targetVer, _, err := ParseVersionOrRange(targetVersion)
	if err != nil {
		return "", fmt.Errorf("invalid target version: %w", err)
	}

	// Parse existing version/range
	existingIsVer, _, existingRange, err := ParseVersionOrRange(existingVersion)
	if err != nil {
		// If existing version is invalid, use target as is
		return targetVersion, nil
	}

	// If existing is a range
	if !existingIsVer && existingRange != nil {
		if targetIsVer {
			// If target is exact version, check if it fits in existing range
			if existingRange.Check(targetVer) {
				// Keep existing range if target version fits
				return existingVersion, nil
			}
			// Create new range based on target version's major
			nextMajor := targetVer.Major() + 1
			return fmt.Sprintf(">= %d, < %d", targetVer.Major(), nextMajor), nil
		}
		// If target is also a range, use target range
		return targetVersion, nil
	}

	// If existing is exact, keep using exact versions
	if existingIsVer {
		if targetIsVer {
			// Both are exact, use target
			return targetVersion, nil
		}
		// Target is range but we want exact, convert it
		return ConvertToExactVersion(targetVersion)
	}

	// Fallback to target as is
	return targetVersion, nil
}
