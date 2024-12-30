package version

import (
	"strconv"
	"strings"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// normalizeVersionString ensures consistent formatting of version strings
func normalizeVersionString(version string) string {
	// Handle complex ranges with OR
	if strings.Contains(version, "||") {
		parts := strings.Split(version, "||")
		for i, part := range parts {
			parts[i] = normalizeVersionString(strings.TrimSpace(part))
		}
		return strings.Join(parts, " || ")
	}

	// Remove all spaces first
	version = strings.ReplaceAll(version, " ", "")

	// Add spaces after operators and commas
	version = strings.ReplaceAll(version, ">=", ">= ")
	version = strings.ReplaceAll(version, "<=", "<= ")
	version = strings.ReplaceAll(version, ">", "> ")
	version = strings.ReplaceAll(version, "<", "< ")
	version = strings.ReplaceAll(version, ",", ", ")

	// Fix any incorrect spacing
	version = strings.ReplaceAll(version, "> =", ">=")
	version = strings.ReplaceAll(version, "< =", "<=")

	// Remove extra spaces
	version = strings.TrimSpace(version)
	for strings.Contains(version, "  ") {
		version = strings.ReplaceAll(version, "  ", " ")
	}

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
