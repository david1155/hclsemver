package version

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input       string
		wantIsVer   bool
		wantVersion string
		wantErr     bool
	}{
		// Single versions
		{"1.2.3", true, "1.2.3", false},
		{"1.2.3-alpha.1+build123", true, "1.2.3-alpha.1+build123", false},
		{"v2.0.0", true, "2.0.0", false}, // 'v' prefix stripped by .String()
		{"invalid_version", false, "", true},

		// Ranges (constraints)
		{">=1.0.0,<2.0.0", false, "", false},
		{"^1.5.0", false, "", false},
		{"~>3.1.2", false, "", false},
		{"~>3", false, "", false},

		// Spaces in operators
		{">= 1, < 2", false, "", false}, // after we remove spaces, => ">=1,<2"
	}

	for _, tc := range tests {
		isVer, ver, rng, err := ParseVersionOrRange(tc.input)

		if tc.wantErr && err == nil {
			t.Errorf("input=%q expected error, got none", tc.input)
			continue
		}
		if !tc.wantErr && err != nil {
			t.Errorf("input=%q unexpected error: %v", tc.input, err)
			continue
		}

		if isVer != tc.wantIsVer {
			t.Errorf("input=%q isVersion=%v, want %v", tc.input, isVer, tc.wantIsVer)
		}

		if !tc.wantErr && isVer && ver != nil {
			gotVerString := ver.String()
			if gotVerString != tc.wantVersion {
				t.Errorf("input=%q ver.String()=%q, want %q",
					tc.input, gotVerString, tc.wantVersion)
			}
			if rng != nil {
				t.Errorf("input=%q expected constraints=nil, got %v", tc.input, rng)
			}
		}
	}
}

func TestRangesOverlap(t *testing.T) {
	cases := []struct {
		a             string
		b             string
		expectOverlap bool
	}{
		{">=1.0.0,<2.0.0", ">=1.5.0,<1.6.0", true},
		{">=2.0.0,<3.0.0", ">=3.0.0,<4.0.0", false},
		{"~>3", ">=2.0.0,<4.0.0", true},
		{"^1.2.3", "~1.2", true},
		{">1.0.0 <1.2.0 || >=2.0.0 <2.1.0", "1.x", true},
	}

	for _, tc := range cases {
		aConstr, errA := semver.NewConstraint(tc.a)
		bConstr, errB := semver.NewConstraint(tc.b)
		if errA != nil || errB != nil {
			t.Fatalf("parse error: a=%v errA=%v, b=%v errB=%v", tc.a, errA, tc.b, errB)
		}
		got := RangesOverlap(aConstr, bConstr)
		if got != tc.expectOverlap {
			t.Errorf("RangesOverlap(%q, %q) = %v, want %v",
				tc.a, tc.b, got, tc.expectOverlap)
		}
	}
}

func TestDecideVersionOrRange(t *testing.T) {
	tests := []struct {
		name     string
		oldInput string
		newInput string
		want     string
	}{
		// both single
		{"single vs single: new higher", "1.2.3", "2.1.0", "2.1.0"},
		{"single vs single: old higher", "2.2.1", "2.0.0", "2.2.1"},
		{"single vs single: equal => keep old", "3.0.0", "3.0.0", "3.0.0"},

		// old single, new range
		{"single vs range: fits", "1.5.0", ">=1.0.0,<2.0.0", "1.5.0"},
		{"single vs range: old higher than range max", "2.5.0", ">=1.0.0,<2.0.0", "2.5.0"},
		{"single vs range: doesn't fit", "2.5.0", ">=3.0.0,<4.0.0", ">=3.0.0,<4.0.0"},

		// old range, new single
		{"range vs single: fits", ">=1.0.0,<2.0.0", "1.5.0", ">=1.0.0,<2.0.0"},
		{"range vs single: range max higher than new", ">=2.0.0,<3.0.0", "1.5.0", ">=2.0.0,<3.0.0"},
		{"range vs single: doesn't fit", ">=1.0.0,<2.0.0", "2.5.0", "2.5.0"},

		// both range
		{"range vs range: overlap", ">=1.0.0,<2.0.0", ">=1.5.0,<2.5.0", ">=1.0.0,<2.0.0"},
		{"range vs range: no overlap", ">=1.0.0,<2.0.0", ">=3.0.0,<4.0.0", ">=3.0.0,<4.0.0"},

		// Backward version protection tests
		{"protect: single vs single backward", "2.0.0", "1.0.0", "2.0.0"},
		{"protect: single vs range backward", "2.0.0", ">=1.0.0,<1.5.0", "2.0.0"},
		{"protect: range vs single backward", ">=2.0.0,<3.0.0", "1.5.0", ">=2.0.0,<3.0.0"},
		{"protect: range vs range backward", ">=2.0.0,<3.0.0", ">=1.0.0,<2.0.0", ">=2.0.0,<3.0.0"},

		// Dynamic strategy tests
		{"dynamic: backward protection - range with higher minimum", ">= 3.2.2, < 4", "3.2.1", ">= 3.2.2, < 4"},
		{"dynamic: backward protection - range with higher minimum (complex)", ">= 3.2.0, < 4.0.0", "3.0.0", ">= 3.2.0, < 4.0.0"},
		{"dynamic: backward protection - range with same minimum", ">= 3.2.0, < 4.0.0", "3.2.0", ">= 3.2.0, < 4.0.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldIsVer, oldVer, oldRange, err := ParseVersionOrRange(tc.oldInput)
			if err != nil {
				t.Fatalf("parse old=%q error: %v", tc.oldInput, err)
			}
			newIsVer, newVer, newRange, err := ParseVersionOrRange(tc.newInput)
			if err != nil {
				t.Fatalf("parse new=%q error: %v", tc.newInput, err)
			}

			got := DecideVersionOrRange(oldIsVer, oldVer, oldRange, tc.oldInput,
				newIsVer, newVer, newRange, tc.newInput)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExpandTerraformTildeArrow(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"~>1.2.3", ">=1.2.3, <2.0.0"},
		{"~>2.0", ">=2.0.0, <3.0.0"},
		{"~>3", ">=3.0.0, <4.0.0"},
		{"1.2.3", "1.2.3"},
		{">=1.0.0", ">=1.0.0"},
		{"~>1.2.3 || ~>2.0.0", ">=1.2.3, <2.0.0 || >=2.0.0, <3.0.0"},
		{"", ""},
		{"~>INVALID", ">=0.0.0, <1.0.0"},
	}

	for _, tc := range tests {
		got := ExpandTerraformTildeArrow(tc.input)
		if got != tc.expected {
			t.Errorf("ExpandTerraformTildeArrow(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestBuildRangeFromTildePart(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3", ">=1.2.3, <2.0.0"},
		{"2.0", ">=2.0.0, <3.0.0"},
		{"3", ">=3.0.0, <4.0.0"},
		{"", "~>MISSING"},
		{"1.2.3.4", "~>INVALID"},
		{" 1.2.3 ", ">=1.2.3, <2.0.0"}, // test trimming
	}

	for _, tc := range tests {
		got := buildRangeFromTildePart(tc.input)
		if got != tc.expected {
			t.Errorf("buildRangeFromTildePart(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestReadToken(t *testing.T) {
	tests := []struct {
		input          string
		expectedToken  string
		expectedRemain string
	}{
		{"1.2.3", "1.2.3", ""},
		{"1.2.3 || 2.0.0", "1.2.3", " || 2.0.0"},
		{"1.2.3,2.0.0", "1.2.3", ",2.0.0"},
		{"1.2.3 2.0.0", "1.2.3", " 2.0.0"},
		{"", "", ""},
		{" 1.2.3", "", " 1.2.3"},
		{"1.2.3||2.0.0", "1.2.3", "||2.0.0"},
	}

	for _, tc := range tests {
		gotToken, gotRemain := readToken(tc.input)
		if gotToken != tc.expectedToken || gotRemain != tc.expectedRemain {
			t.Errorf("readToken(%q) = (%q, %q), want (%q, %q)",
				tc.input, gotToken, gotRemain, tc.expectedToken, tc.expectedRemain)
		}
	}
}

func TestVersionStrategies(t *testing.T) {
	tests := []struct {
		name            string
		strategy        Strategy
		targetVersion   string
		existingVersion string
		want            string
		wantErr         bool
	}{
		// Dynamic strategy tests
		{
			name:            "dynamic: existing exact -> new exact",
			strategy:        StrategyDynamic,
			targetVersion:   "2.0.0",
			existingVersion: "1.0.0",
			want:            "2.0.0",
		},
		{
			name:            "dynamic: existing range -> keep range (target within)",
			strategy:        StrategyDynamic,
			targetVersion:   "2.0.0",
			existingVersion: ">= 1.0.0, < 3.0.0",
			want:            ">= 1.0.0, < 3.0.0",
		},
		{
			name:            "dynamic: existing range -> new range (target outside)",
			strategy:        StrategyDynamic,
			targetVersion:   "3.1.0",
			existingVersion: ">= 2.0.0, < 3",
			want:            ">= 3, < 4",
		},
		{
			name:            "dynamic: no existing -> exact",
			strategy:        StrategyDynamic,
			targetVersion:   "2.0.0",
			existingVersion: "",
			want:            "2.0.0",
		},
		{
			name:            "dynamic: backward protection - range with higher minimum",
			strategy:        StrategyDynamic,
			targetVersion:   "3.2.1",
			existingVersion: ">= 3.2.2, < 4",
			want:            ">= 3.2.2, < 4",
		},
		{
			name:            "dynamic: backward protection - exact to lower range",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 2.0.0, < 3.0.0",
			existingVersion: "3.1.0",
			want:            "3.1.0",
		},
		{
			name:            "dynamic: backward protection - range with patch version",
			strategy:        StrategyDynamic,
			targetVersion:   "3.2.0",
			existingVersion: ">= 3.2.5, < 4.0.0",
			want:            ">= 3.2.5, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - complex range to simpler range",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 3, < 4",
			existingVersion: ">= 3.2.5, < 3.3.0",
			want:            ">= 3.2.5, < 3.3.0",
		},
		{
			name:            "dynamic: backward protection - tilde arrow to range",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 2.0.0, < 3.0.0",
			existingVersion: "~> 3.2",
			want:            ">= 3.2.0, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - range with spaces",
			strategy:        StrategyDynamic,
			targetVersion:   "3.0.0",
			existingVersion: ">= 3.2.0,< 4.0.0",  // no space after comma
			want:            ">= 3.2.0, < 4.0.0", // normalized with space
		},
		{
			name:            "dynamic: backward protection - range with pre-release",
			strategy:        StrategyDynamic,
			targetVersion:   "3.0.0",
			existingVersion: ">= 3.2.0-beta.1, < 4.0.0",
			want:            ">= 3.2.0-beta.1, < 4.0.0",
		},
		{
			name:            "dynamic: exact version to higher exact",
			strategy:        StrategyDynamic,
			targetVersion:   "3.5.0",
			existingVersion: "3.2.1",
			want:            "3.5.0",
		},
		{
			name:            "dynamic: range to higher exact within range",
			strategy:        StrategyDynamic,
			targetVersion:   "3.2.5",
			existingVersion: ">= 3.2.0, < 4.0.0",
			want:            ">= 3.2.0, < 4.0.0",
		},
		{
			name:            "dynamic: range to exact outside range",
			strategy:        StrategyDynamic,
			targetVersion:   "4.0.0",
			existingVersion: ">= 3.2.0, < 4.0.0",
			want:            ">= 4, < 5",
		},
		// Additional backward protection test cases
		{
			name:            "dynamic: backward protection - overlapping ranges with higher minimum",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 3.0.0, < 4.0.0",
			existingVersion: ">= 3.2.0, < 4.0.0",
			want:            ">= 3.2.0, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - non-overlapping ranges",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 2.0.0, < 3.0.0",
			existingVersion: ">= 3.0.0, < 4.0.0",
			want:            ">= 3.0.0, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - exact to lower exact",
			strategy:        StrategyDynamic,
			targetVersion:   "2.0.0",
			existingVersion: "3.0.0",
			want:            "3.0.0",
		},
		{
			name:            "dynamic: backward protection - range to lower range",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 2.0.0, < 3.0.0",
			existingVersion: ">= 3.0.0, < 4.0.0",
			want:            ">= 3.0.0, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - complex range with pre-release",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 3.0.0, < 4.0.0",
			existingVersion: ">= 3.2.0-beta.1, < 3.3.0-rc.1",
			want:            ">= 3.2.0-beta.1, < 3.3.0-rc.1",
		},
		{
			name:            "dynamic: backward protection - tilde arrow with higher version",
			strategy:        StrategyDynamic,
			targetVersion:   "~> 3.1",
			existingVersion: "~> 3.2",
			want:            ">= 3.2.0, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - range with higher patch version",
			strategy:        StrategyDynamic,
			targetVersion:   ">= 3.2.4, < 4.0.0",
			existingVersion: ">= 3.2.5, < 4.0.0",
			want:            ">= 3.2.5, < 4.0.0",
		},
		{
			name:            "dynamic: backward protection - exact to range with higher minimum",
			strategy:        StrategyDynamic,
			targetVersion:   "3.1.0",
			existingVersion: ">= 3.2.0, < 4.0.0",
			want:            ">= 3.2.0, < 4.0.0",
		},

		// Exact strategy tests
		{
			name:            "exact: from exact",
			strategy:        StrategyExact,
			targetVersion:   "2.0.0",
			existingVersion: "1.0.0",
			want:            "2.0.0",
		},
		{
			name:            "exact: from range - should error",
			strategy:        StrategyExact,
			targetVersion:   ">=2.0.0, <3.0.0",
			existingVersion: "1.0.0",
			wantErr:         true,
		},
		{
			name:            "exact: invalid version",
			strategy:        StrategyExact,
			targetVersion:   "invalid",
			existingVersion: "1.0.0",
			wantErr:         true,
		},

		// Range strategy tests
		{
			name:            "range: from exact",
			strategy:        StrategyRange,
			targetVersion:   "2.0.0",
			existingVersion: "1.0.0",
			want:            ">= 2.0.0, < 3.0.0",
		},
		{
			name:            "range: from range",
			strategy:        StrategyRange,
			targetVersion:   ">= 2.0.0, < 3.0.0",
			existingVersion: ">= 1.0.0, < 2.0.0",
			want:            ">= 2.0.0, < 3.0.0",
		},
		{
			name:            "range: invalid version",
			strategy:        StrategyRange,
			targetVersion:   "invalid",
			existingVersion: "1.0.0",
			wantErr:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ApplyVersionStrategy(tc.strategy, tc.targetVersion, tc.existingVersion)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
