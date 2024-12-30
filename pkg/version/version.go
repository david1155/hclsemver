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

// findHighestVersionInRange tries to find the highest version that satisfies the constraints
// using binary search for better performance O(log n)
func findHighestVersionInRange(c *semver.Constraints) *semver.Version {
	if c == nil {
		return nil
	}

	// Binary search for major version
	left, right := 0, MAX_MAJOR
	var maxMajor int = -1

	for left <= right {
		major := (left + right) / 2
		// Try with maximum minor and patch
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, MAX_MINOR, MAX_PATCH))

		if c.Check(testVer) {
			maxMajor = major
			// Try higher major version
			left = major + 1
		} else {
			// Try lower major version
			right = major - 1
		}
	}

	if maxMajor == -1 {
		return nil
	}

	// Binary search for minor version
	left, right = 0, MAX_MINOR
	var maxMinor int = -1

	for left <= right {
		minor := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, minor, MAX_PATCH))

		if c.Check(testVer) {
			maxMinor = minor
			// Try higher minor version
			left = minor + 1
		} else {
			// Try lower minor version
			right = minor - 1
		}
	}

	if maxMinor == -1 {
		// Try with minor 0 if no other minor version works
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.0.%d", maxMajor, MAX_PATCH))
		if c.Check(testVer) {
			maxMinor = 0
		} else {
			// If no minor version works with max patch, try with patch 0
			testVer, _ = semver.NewVersion(fmt.Sprintf("%d.0.0", maxMajor))
			if c.Check(testVer) {
				return testVer
			}
			return nil
		}
	}

	// Binary search for patch version
	left, right = 0, MAX_PATCH
	var maxPatch int = -1

	for left <= right {
		patch := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, patch))

		if c.Check(testVer) {
			maxPatch = patch
			// Try higher patch version
			left = patch + 1
		} else {
			// Try lower patch version
			right = patch - 1
		}
	}

	if maxPatch == -1 {
		// Try with patch 0 if no other patch version works
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", maxMajor, maxMinor))
		if c.Check(testVer) {
			maxPatch = 0
		} else {
			return nil
		}
	}

	// Create and return the final version
	finalVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, maxPatch))
	return finalVer
}

// findLowestVersionInRange tries to find the lowest version that satisfies the constraints
func findLowestVersionInRange(c *semver.Constraints) *semver.Version {
	if c == nil {
		return nil
	}

	var lowestVer *semver.Version

	// Binary search for major version
	left, right := 0, MAX_MAJOR
	var minMajor int = -1

	for left <= right {
		major := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", major))

		// Check if this major version satisfies any of the constraints
		if c.Check(testVer) {
			minMajor = major
			// Try lower major version
			right = major - 1
		} else {
			// Try higher major version
			left = major + 1
		}
	}

	if minMajor == -1 {
		// Try to find any valid version by linear search
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

	// Binary search for minor version
	left, right = 0, MAX_MINOR
	var minMinor int = -1

	for left <= right {
		minor := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", minMajor, minor))

		if c.Check(testVer) {
			minMinor = minor
			// Try lower minor version
			right = minor - 1
		} else {
			// Try higher minor version
			left = minor + 1
		}
	}

	if minMinor == -1 {
		// Try with minor 0 if no other minor version works
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", minMajor))
		if c.Check(testVer) {
			minMinor = 0
		} else {
			// If no minor version works, try next major version
			nextMajorVer, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", minMajor+1))
			if c.Check(nextMajorVer) {
				return nextMajorVer
			}
			return nil
		}
	}

	// Binary search for patch version
	left, right = 0, MAX_PATCH
	var minPatch int = -1

	for left <= right {
		patch := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", minMajor, minMinor, patch))

		if c.Check(testVer) {
			minPatch = patch
			// Try lower patch version
			right = patch - 1
		} else {
			// Try higher patch version
			left = patch + 1
		}
	}

	if minPatch == -1 {
		// Try with patch 0 if no other patch version works
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", minMajor, minMinor))
		if c.Check(testVer) {
			minPatch = 0
		} else {
			// If no patch version works, try next minor version
			nextMinorVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", minMajor, minMinor+1))
			if c.Check(nextMinorVer) {
				return nextMinorVer
			}
			return nil
		}
	}

	// Create and return the final version
	finalVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", minMajor, minMinor, minPatch))
	return finalVer
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

// NormalizeVersionString ensures consistent formatting of version strings
func NormalizeVersionString(version string) string {
	// Remove all spaces first
	version = strings.ReplaceAll(version, " ", "")

	// Add commas between operators if missing
	version = strings.ReplaceAll(version, ">=", ">= ")
	version = strings.ReplaceAll(version, "<", ", <")

	// Remove extra commas
	version = strings.ReplaceAll(version, ",,", ",")
	version = strings.ReplaceAll(version, ", ,", ",")

	// Add spaces after commas
	version = strings.ReplaceAll(version, ",", ", ")

	// Remove any trailing spaces
	version = strings.TrimSpace(version)

	// Remove leading comma if present
	if strings.HasPrefix(version, ", ") {
		version = strings.TrimPrefix(version, ", ")
	}

	return version
}
