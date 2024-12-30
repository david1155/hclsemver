package version

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// findHighestVersionInRange tries to find the highest version that satisfies the constraints
// using binary search for better performance O(log n)
func findHighestVersionInRange(c *semver.Constraints) *semver.Version {
	if c == nil {
		return nil
	}

	// Try strategic points first for quick exit
	strategicPoints := []struct{ major, minor, patch uint64 }{
		{major: MAX_MAJOR, minor: MAX_MINOR, patch: MAX_PATCH},             // Maximum possible
		{major: MAX_MAJOR / 2, minor: MAX_MINOR / 2, patch: MAX_PATCH / 2}, // Mid-range
		{major: MAX_MAJOR / 4, minor: MAX_MINOR / 4, patch: MAX_PATCH / 4}, // Quarter-range
	}

	var highestVer *semver.Version
	for _, p := range strategicPoints {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", p.major, p.minor, p.patch))
		if c.Check(testVer) {
			highestVer = testVer
			break
		}
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
		// If no major version found with max minor/patch, try with lower values
		for major := MAX_MAJOR; major >= 0; major-- {
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", major))
			if c.Check(testVer) {
				maxMajor = major
				break
			}
		}
		if maxMajor == -1 {
			return highestVer // Return the highest version found in strategic points, if any
		}
	}

	// Binary search for minor version with optimized range
	left, right = 0, MAX_MINOR
	var maxMinor int = -1

	// Try strategic minor versions first
	strategicMinors := []int{MAX_MINOR, MAX_MINOR / 2, MAX_MINOR / 4}
	for _, minor := range strategicMinors {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, minor, MAX_PATCH))
		if c.Check(testVer) {
			maxMinor = minor
			break
		}
	}

	if maxMinor == -1 {
		// Binary search for minor version
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
			return highestVer // Return the highest version found in strategic points, if any
		}
	}

	// Binary search for patch version with optimized range
	left, right = 0, MAX_PATCH
	var maxPatch int = -1

	// Try strategic patch versions first
	strategicPatches := []int{MAX_PATCH, MAX_PATCH / 2, MAX_PATCH / 4}
	for _, patch := range strategicPatches {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, patch))
		if c.Check(testVer) {
			maxPatch = patch
			break
		}
	}

	if maxPatch == -1 {
		// Binary search for patch version
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
	}

	if maxPatch == -1 {
		// Try with patch 0 if no other patch version works
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", maxMajor, maxMinor))
		if c.Check(testVer) {
			maxPatch = 0
		} else {
			return highestVer // Return the highest version found in strategic points, if any
		}
	}

	// Create and return the final version
	finalVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", maxMajor, maxMinor, maxPatch))
	if highestVer != nil && highestVer.GreaterThan(finalVer) {
		return highestVer
	}
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
	if a == nil || b == nil {
		return false
	}

	// Quick boundary check
	aMin := findLowestVersionInRange(a)
	aMax := findHighestVersionInRange(a)
	bMin := findLowestVersionInRange(b)
	bMax := findHighestVersionInRange(b)

	// If either range is empty, they can't overlap
	if aMin == nil || aMax == nil || bMin == nil || bMax == nil {
		// Fall back to linear search for complex cases
		return linearSearchOverlap(a, b)
	}

	// If one range is completely before or after the other, they can't overlap
	if aMax.LessThan(bMin) || bMax.LessThan(aMin) {
		return false
	}

	// For ranges that might overlap, check strategic points
	// Check the boundaries
	if a.Check(bMin) || a.Check(bMax) || b.Check(aMin) || b.Check(aMax) {
		return true
	}

	// Check the midpoint of the overlapping region
	if aMin.LessThan(bMax) && bMin.LessThan(aMax) {
		// Calculate multiple strategic points in the overlapping region
		points := []struct{ major, minor, patch uint64 }{
			{
				major: (aMin.Major() + bMax.Major()) / 2,
				minor: (aMin.Minor() + bMax.Minor()) / 2,
				patch: (aMin.Patch() + bMax.Patch()) / 2,
			},
			{
				major: aMin.Major(),
				minor: bMax.Minor(),
				patch: (aMin.Patch() + bMax.Patch()) / 2,
			},
			{
				major: bMin.Major(),
				minor: aMax.Minor(),
				patch: (aMin.Patch() + bMax.Patch()) / 2,
			},
		}

		// Try each strategic point
		for _, p := range points {
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", p.major, p.minor, p.patch))
			if a.Check(testVer) && b.Check(testVer) {
				return true
			}
		}

		// Try quarter points in the overlapping region
		quarterMajor := (aMin.Major()*3 + bMax.Major()) / 4
		quarterMinor := (aMin.Minor()*3 + bMax.Minor()) / 4
		quarterPatch := (aMin.Patch()*3 + bMax.Patch()) / 4
		quarterVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", quarterMajor, quarterMinor, quarterPatch))

		threeQuarterMajor := (aMin.Major() + bMax.Major()*3) / 4
		threeQuarterMinor := (aMin.Minor() + bMax.Minor()*3) / 4
		threeQuarterPatch := (aMin.Patch() + bMax.Patch()*3) / 4
		threeQuarterVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", threeQuarterMajor, threeQuarterMinor, threeQuarterPatch))

		if (a.Check(quarterVer) && b.Check(quarterVer)) || (a.Check(threeQuarterVer) && b.Check(threeQuarterVer)) {
			return true
		}
	}

	// For complex cases or potential overlaps, use binary search with reduced range
	start := max(0, min(int(aMin.Major()), int(bMin.Major())))
	end := min(MAX_MAJOR, max(int(aMax.Major()), int(bMax.Major())))

	// Try strategic points at each major version
	for major := start; major <= end; major++ {
		// Try strategic minor versions
		strategicMinors := []int{0, MAX_MINOR / 4, MAX_MINOR / 2, (MAX_MINOR * 3) / 4, MAX_MINOR}
		for _, minor := range strategicMinors {
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", major, minor))
			if a.Check(testVer) && b.Check(testVer) {
				return true
			}
		}

		// Binary search for minor version if strategic points didn't find overlap
		left, right := 0, MAX_MINOR
		for left <= right {
			minor := (left + right) / 2
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", major, minor))

			if a.Check(testVer) && b.Check(testVer) {
				return true
			}

			// Try both ranges to determine search direction
			aValid := a.Check(testVer)
			bValid := b.Check(testVer)

			if !aValid && !bValid {
				// If neither range accepts this minor version, try higher
				left = minor + 1
			} else if aValid || bValid {
				// If at least one range accepts this minor version, try patch versions
				if searchPatchVersions(a, b, major, minor) {
					return true
				}
				// Try lower minor version as well
				right = minor - 1
			} else {
				// Try both directions by splitting the search
				if minor > 0 && searchMinorRange(a, b, major, 0, minor-1) {
					return true
				}
				if minor < MAX_MINOR && searchMinorRange(a, b, major, minor+1, MAX_MINOR) {
					return true
				}
				break
			}
		}
	}

	// Final check: try a small linear search around the boundaries
	boundaryRange := 2 // Check 2 versions before and after each boundary
	for _, major := range []int{start - boundaryRange, start, end, end + boundaryRange} {
		if major < 0 || major > MAX_MAJOR {
			continue
		}
		for minor := 0; minor <= min(5, MAX_MINOR); minor++ {
			for patch := 0; patch <= min(5, MAX_PATCH); patch++ {
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
				if a.Check(testVer) && b.Check(testVer) {
					return true
				}
			}
		}
	}

	return false
}
