package version

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// searchMinorRangeLinear searches a specific range of minor versions with optimized search
func searchMinorRangeLinear(a, b *semver.Constraints, major, start, end int) bool {
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

	// Try strategic points first for quick exit
	strategicPoints := []struct {
		minor int
		patch int
		desc  string
	}{
		{minor: start, patch: 0, desc: "start"},                                  // Start
		{minor: end, patch: MAX_PATCH, desc: "end"},                              // End
		{minor: (start + end) / 2, patch: MAX_PATCH / 2, desc: "mid"},            // Middle
		{minor: start + (end-start)/4, patch: MAX_PATCH / 4, desc: "quarter"},    // Quarter
		{minor: end - (end-start)/4, patch: MAX_PATCH * 3 / 4, desc: "3quarter"}, // Three-quarter
		{minor: start + (end-start)/8, patch: MAX_PATCH / 8, desc: "eighth"},     // Eighth
		{minor: end - (end-start)/8, patch: MAX_PATCH * 7 / 8, desc: "7eighth"},  // Seven-eighth
	}

	// Try strategic points with caching
	for _, p := range strategicPoints {
		if checkVersion(major, p.minor, p.patch, true) && checkVersion(major, p.minor, p.patch, false) {
			return true
		}
	}

	// Quick boundary check with caching
	aStartValid := checkVersion(major, start, 0, true)
	aEndValid := checkVersion(major, end, MAX_PATCH, true)
	bStartValid := checkVersion(major, start, 0, false)
	bEndValid := checkVersion(major, end, MAX_PATCH, false)

	// Early exit if no overlap possible
	if (!aStartValid && !aEndValid) || (!bStartValid && !bEndValid) {
		return false
	}

	// Optimize search range based on boundary checks
	left := start
	right := end
	if !aStartValid && !bStartValid {
		// Both ranges reject low versions, start higher
		left = start + (end-start)/4
	}
	if !aEndValid && !bEndValid {
		// Both ranges reject high versions, end lower
		right = end - (end-start)/4
	}

	// Binary search with optimized range and caching
	for left <= right {
		minor := (left + right) / 2

		// Try strategic patch points for this minor version
		patchPoints := []int{0, MAX_PATCH / 4, MAX_PATCH / 2, MAX_PATCH * 3 / 4, MAX_PATCH}
		foundOverlap := false
		for _, patch := range patchPoints {
			if checkVersion(major, minor, patch, true) && checkVersion(major, minor, patch, false) {
				foundOverlap = true
				break
			}
		}
		if foundOverlap {
			return true
		}

		// Check if either range accepts this minor version
		aValid := false
		bValid := false
		for _, patch := range patchPoints {
			if checkVersion(major, minor, patch, true) {
				aValid = true
			}
			if checkVersion(major, minor, patch, false) {
				bValid = true
			}
			if aValid && bValid {
				break
			}
		}

		// If both ranges accept versions in this minor version, check patch versions
		if aValid && bValid {
			if searchPatchVersionsLinear(a, b, major, minor) {
				return true
			}
		}

		// Check nearby versions for potential overlap
		if minor > start {
			prevMinor := minor - 1
			if searchPatchVersionsLinear(a, b, major, prevMinor) {
				return true
			}
		}
		if minor < end {
			nextMinor := minor + 1
			if searchPatchVersionsLinear(a, b, major, nextMinor) {
				return true
			}
		}

		// Determine search direction based on acceptance patterns
		if !aValid && !bValid {
			// Neither range accepts this version, try higher
			left = minor + 1
		} else if (aValid && !bValid && bEndValid) || (!aValid && bValid && aEndValid) {
			// One range accepts this version and the other accepts higher versions
			left = minor + 1
		} else {
			// Try lower versions
			right = minor - 1
		}

		// Try strategic jumps if we're getting close
		if right-left <= 5 {
			// Check all versions in between
			for m := left; m <= right; m++ {
				if searchPatchVersionsLinear(a, b, major, m) {
					return true
				}
			}
			break
		}
	}

	return false
}
