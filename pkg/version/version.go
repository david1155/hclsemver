package version

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// searchMinorRange searches a specific range of minor versions using binary search
func searchMinorRange(a, b *semver.Constraints, major, start, end int) bool {
	// Quick boundary check
	testStart, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", major, start))
	testEnd, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, end, MAX_PATCH))

	// Check boundaries first
	if (a.Check(testStart) && b.Check(testStart)) || (a.Check(testEnd) && b.Check(testEnd)) {
		return true
	}

	// Try strategic points for quick exit
	strategicPoints := []struct{ minor, patch int }{
		{minor: (start + end) / 2, patch: MAX_PATCH / 2},       // Middle point
		{minor: start + (end-start)/4, patch: MAX_PATCH / 4},   // Quarter point
		{minor: end - (end-start)/4, patch: MAX_PATCH * 3 / 4}, // Three-quarter point
	}

	for _, p := range strategicPoints {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, p.minor, p.patch))
		if a.Check(testVer) && b.Check(testVer) {
			return true
		}
	}

	// Check if either range accepts any version in this range
	aStartValid := a.Check(testStart)
	aEndValid := a.Check(testEnd)
	bStartValid := b.Check(testStart)
	bEndValid := b.Check(testEnd)

	// If either range doesn't accept any version in this range, return false
	if (!aStartValid && !aEndValid) || (!bStartValid && !bEndValid) {
		return false
	}

	// If both ranges accept versions in this range, use binary search
	left, right := start, end
	for left <= right {
		minor := (left + right) / 2

		// Try with strategic patch versions for this minor version
		patchPoints := []int{0, MAX_PATCH / 4, MAX_PATCH / 2, (MAX_PATCH * 3) / 4, MAX_PATCH}
		for _, patch := range patchPoints {
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
			if a.Check(testVer) && b.Check(testVer) {
				return true
			}
		}

		// If strategic points didn't find overlap, check full patch range
		testMin, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", major, minor))
		testMax, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, MAX_PATCH))

		aAcceptsMin := a.Check(testMin)
		aAcceptsMax := a.Check(testMax)
		bAcceptsMin := b.Check(testMin)
		bAcceptsMax := b.Check(testMax)

		// Quick exit if both ranges accept the same version
		if (aAcceptsMin && bAcceptsMin) || (aAcceptsMax && bAcceptsMax) {
			return true
		}

		// If both ranges accept versions in this minor version
		if (aAcceptsMin || aAcceptsMax) && (bAcceptsMin || bAcceptsMax) {
			// Check patch versions
			if searchPatchVersions(a, b, major, minor) {
				return true
			}
		}

		// Determine search direction based on acceptance patterns
		if !aAcceptsMax && !bAcceptsMax {
			// Neither range accepts high versions, try lower
			right = minor - 1
		} else if !aAcceptsMin && !bAcceptsMin {
			// Neither range accepts low versions, try higher
			left = minor + 1
		} else {
			// Complex case: split search into two parts
			// Only search if there's a reasonable chance of overlap
			if minor > start && (aAcceptsMin || aAcceptsMax) && (bAcceptsMin || bAcceptsMax) {
				if searchMinorRange(a, b, major, start, minor-1) {
					return true
				}
			}
			if minor < end && (aAcceptsMin || aAcceptsMax) && (bAcceptsMin || bAcceptsMax) {
				if searchMinorRange(a, b, major, minor+1, end) {
					return true
				}
			}
			break
		}
	}

	return false
}

// searchPatchVersions uses binary search to find overlapping patch versions
func searchPatchVersions(a, b *semver.Constraints, major, minor int) bool {
	// Try strategic points first for quick exit
	strategicPoints := []int{
		0,                   // Start
		MAX_PATCH / 4,       // Quarter
		MAX_PATCH / 2,       // Middle
		(MAX_PATCH * 3) / 4, // Three-quarter
		MAX_PATCH,           // End
	}

	for _, patch := range strategicPoints {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
		if a.Check(testVer) && b.Check(testVer) {
			return true
		}
	}

	// Quick boundary check
	testMin, _ := semver.NewVersion(fmt.Sprintf("%d.%d.0", major, minor))
	testMax, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, MAX_PATCH))

	// Check range acceptance patterns
	aMinValid := a.Check(testMin)
	aMaxValid := a.Check(testMax)
	bMinValid := b.Check(testMin)
	bMaxValid := b.Check(testMax)

	// Early exit if no overlap possible
	if (!aMinValid && !aMaxValid) || (!bMinValid && !bMaxValid) {
		return false
	}

	// Determine search range based on acceptance patterns
	left := 0
	right := MAX_PATCH
	if !aMinValid && !bMinValid {
		// Both ranges reject low versions, start higher
		left = MAX_PATCH / 4
	}
	if !aMaxValid && !bMaxValid {
		// Both ranges reject high versions, end lower
		right = (MAX_PATCH * 3) / 4
	}

	// Binary search with optimized range
	for left <= right {
		patch := (left + right) / 2
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))

		aValid := a.Check(testVer)
		bValid := b.Check(testVer)

		if aValid && bValid {
			return true
		}

		// Determine search direction based on acceptance patterns
		if !aValid && !bValid {
			// Neither range accepts this version, try higher
			left = patch + 1
		} else if (aValid && !bValid && bMaxValid) || (!aValid && bValid && aMaxValid) {
			// One range accepts this version and the other accepts higher versions
			left = patch + 1
		} else {
			// Try lower versions
			right = patch - 1
		}

		// Quick check nearby versions for potential overlap
		if patch > 0 {
			nearbyVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch-1))
			if a.Check(nearbyVer) && b.Check(nearbyVer) {
				return true
			}
		}
		if patch < MAX_PATCH {
			nearbyVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch+1))
			if a.Check(nearbyVer) && b.Check(nearbyVer) {
				return true
			}
		}
	}

	return false
}

// searchPatchVersionsLinear searches for overlapping patch versions with optimized search
func searchPatchVersionsLinear(a, b *semver.Constraints, major, minor int) bool {
	// Try strategic points first for quick exit
	strategicPoints := []struct {
		patch int
		desc  string
	}{
		{patch: 0, desc: "start"},                         // Start
		{patch: MAX_PATCH, desc: "end"},                   // End
		{patch: MAX_PATCH / 2, desc: "mid"},               // Middle
		{patch: MAX_PATCH / 4, desc: "quarter"},           // Quarter
		{patch: MAX_PATCH * 3 / 4, desc: "three-quarter"}, // Three-quarter
		{patch: MAX_PATCH / 8, desc: "eighth"},            // Eighth
		{patch: MAX_PATCH * 7 / 8, desc: "seven-eighth"},  // Seven-eighth
	}

	// Cache for version checks to avoid redundant computations
	type versionKey struct {
		major, minor, patch int
		isA                 bool
	}
	cache := make(map[versionKey]bool)

	checkVersion := func(major, minor, patch int, isA bool) bool {
		key := versionKey{major: major, minor: minor, patch: patch, isA: isA}
		if result, ok := cache[key]; ok {
			return result
		}
		ver, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
		var result bool
		if isA {
			result = a.Check(ver)
		} else {
			result = b.Check(ver)
		}
		cache[key] = result
		return result
	}

	// Try strategic points with caching
	for _, p := range strategicPoints {
		if checkVersion(major, minor, p.patch, true) && checkVersion(major, minor, p.patch, false) {
			return true
		}
	}

	// Quick boundary check with caching
	aMinValid := checkVersion(major, minor, 0, true)
	aMaxValid := checkVersion(major, minor, MAX_PATCH, true)
	bMinValid := checkVersion(major, minor, 0, false)
	bMaxValid := checkVersion(major, minor, MAX_PATCH, false)

	// Early exit if no overlap possible
	if (!aMinValid && !aMaxValid) || (!bMinValid && !bMaxValid) {
		return false
	}

	// Optimize search range based on boundary checks
	left := 0
	right := MAX_PATCH
	if !aMinValid && !bMinValid {
		// Both ranges reject low versions, start higher
		left = MAX_PATCH / 4
	}
	if !aMaxValid && !bMaxValid {
		// Both ranges reject high versions, end lower
		right = MAX_PATCH * 3 / 4
	}

	// Binary search with optimized range and caching
	for left <= right {
		patch := (left + right) / 2
		aValid := checkVersion(major, minor, patch, true)
		bValid := checkVersion(major, minor, patch, false)

		if aValid && bValid {
			return true
		}

		// Check nearby versions for potential overlap
		if patch > 0 {
			prevPatch := patch - 1
			if checkVersion(major, minor, prevPatch, true) && checkVersion(major, minor, prevPatch, false) {
				return true
			}
		}
		if patch < MAX_PATCH {
			nextPatch := patch + 1
			if checkVersion(major, minor, nextPatch, true) && checkVersion(major, minor, nextPatch, false) {
				return true
			}
		}

		// Determine search direction based on acceptance patterns
		if !aValid && !bValid {
			// Neither range accepts this version, try higher
			left = patch + 1
		} else if (aValid && !bValid && bMaxValid) || (!aValid && bValid && aMaxValid) {
			// One range accepts this version and the other accepts higher versions
			left = patch + 1
		} else {
			// Try lower versions
			right = patch - 1
		}

		// Try strategic jumps if we're getting close
		if right-left <= 5 {
			// Check all versions in between
			for p := left; p <= right; p++ {
				if checkVersion(major, minor, p, true) && checkVersion(major, minor, p, false) {
					return true
				}
			}
			break
		}
	}

	return false
}

// linearSearchOverlap performs a search for overlapping versions with optimizations
func linearSearchOverlap(a, b *semver.Constraints) bool {
	// Try strategic points first for quick exit
	strategicPoints := []struct{ major, minor, patch uint64 }{
		{major: 0, minor: 0, patch: 0},                                                 // Minimum
		{major: MAX_MAJOR, minor: MAX_MINOR, patch: MAX_PATCH},                         // Maximum
		{major: MAX_MAJOR / 2, minor: MAX_MINOR / 2, patch: MAX_PATCH / 2},             // Middle
		{major: MAX_MAJOR / 4, minor: MAX_MINOR / 4, patch: MAX_PATCH / 4},             // Quarter
		{major: MAX_MAJOR * 3 / 4, minor: MAX_MINOR * 3 / 4, patch: MAX_PATCH * 3 / 4}, // Three-quarter
	}

	for _, p := range strategicPoints {
		testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", p.major, p.minor, p.patch))
		if a.Check(testVer) && b.Check(testVer) {
			return true
		}
	}

	// Try to find valid ranges first using binary search
	left, right := 0, MAX_MAJOR
	var aMinMajor, aMaxMajor, bMinMajor, bMaxMajor int = -1, -1, -1, -1

	// Binary search for major version ranges
	for left <= right {
		major := (left + right) / 2
		testMin, _ := semver.NewVersion(fmt.Sprintf("%d.0.0", major))
		testMax, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, MAX_MINOR, MAX_PATCH))

		aAcceptsMin := a.Check(testMin)
		aAcceptsMax := a.Check(testMax)
		bAcceptsMin := b.Check(testMin)
		bAcceptsMax := b.Check(testMax)

		// Quick check for overlap at boundaries
		if (aAcceptsMin && bAcceptsMin) || (aAcceptsMax && bAcceptsMax) {
			return true
		}

		// Update valid ranges
		if aAcceptsMin || aAcceptsMax {
			if aMinMajor == -1 {
				aMinMajor = major
			}
			aMaxMajor = major
		}
		if bAcceptsMin || bAcceptsMax {
			if bMinMajor == -1 {
				bMinMajor = major
			}
			bMaxMajor = major
		}

		// Determine search direction
		if !aAcceptsMax && !bAcceptsMax {
			right = major - 1
		} else if !aAcceptsMin && !bAcceptsMin {
			left = major + 1
		} else {
			// Check both directions
			if major > 0 {
				leftMajor := major - 1
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", leftMajor, MAX_MINOR/2, MAX_PATCH/2))
				if a.Check(testVer) && b.Check(testVer) {
					return true
				}
			}
			if major < MAX_MAJOR {
				rightMajor := major + 1
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", rightMajor, MAX_MINOR/2, MAX_PATCH/2))
				if a.Check(testVer) && b.Check(testVer) {
					return true
				}
			}
			break
		}
	}

	// If no valid ranges found, try a reduced search space
	if aMinMajor == -1 || bMinMajor == -1 {
		// Try a smaller range for complex constraints
		for major := 0; major <= min(5, MAX_MAJOR); major++ {
			for minor := 0; minor <= min(10, MAX_MINOR); minor++ {
				// Use binary search for patch versions
				left, right := 0, min(10, MAX_PATCH)
				for left <= right {
					patch := (left + right) / 2
					testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
					if a.Check(testVer) && b.Check(testVer) {
						return true
					}
					// Try both directions
					aValid := a.Check(testVer)
					bValid := b.Check(testVer)
					if !aValid && !bValid {
						left = patch + 1
					} else {
						right = patch - 1
					}
				}
			}
		}
		return false
	}

	// For each overlapping major version, use binary search for minor and patch
	start := max(0, min(aMinMajor, bMinMajor))
	end := min(MAX_MAJOR, max(aMaxMajor, bMaxMajor))

	for major := start; major <= end; major++ {
		// Binary search for minor version
		left, right := 0, MAX_MINOR
		for left <= right {
			minor := (left + right) / 2
			testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, MAX_PATCH/2))

			aValid := a.Check(testVer)
			bValid := b.Check(testVer)

			if aValid && bValid {
				return true
			}

			// Check patch versions if either range accepts this minor version
			if (aValid || bValid) && searchPatchVersionsLinear(a, b, major, minor) {
				return true
			}

			// Try both directions
			if !aValid && !bValid {
				left = minor + 1
			} else {
				// Check both directions with binary search
				if minor > 0 && searchMinorRangeLinear(a, b, major, 0, minor-1) {
					return true
				}
				if minor < MAX_MINOR && searchMinorRangeLinear(a, b, major, minor+1, MAX_MINOR) {
					return true
				}
				break
			}
		}
	}

	// Final check: try a small range around the boundaries
	boundaryRange := 2
	for major := max(0, start-boundaryRange); major <= min(MAX_MAJOR, end+boundaryRange); major++ {
		for minor := 0; minor <= min(5, MAX_MINOR); minor++ {
			// Use binary search for patch versions
			left, right := 0, min(5, MAX_PATCH)
			for left <= right {
				patch := (left + right) / 2
				testVer, _ := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
				if a.Check(testVer) && b.Check(testVer) {
					return true
				}
				// Try both directions
				aValid := a.Check(testVer)
				bValid := b.Check(testVer)
				if !aValid && !bValid {
					left = patch + 1
				} else {
					right = patch - 1
				}
			}
		}
	}

	return false
}
