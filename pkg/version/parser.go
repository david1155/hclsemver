package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
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
