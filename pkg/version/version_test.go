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
		{"old single fits new => keep old", "1.2.3", ">=1.0.0,<2.0.0", "1.2.3"},
		{"old single doesn't fit => new range", "1.2.3", ">=2.0.0,<3.0.0", ">=2.0.0,<3.0.0"},

		// old range, new single
		{"new single fits old => keep old", ">=1.0.0,<2.0.0", "1.2.3", ">=1.0.0,<2.0.0"},
		{"new single doesn't fit => new single", ">=2.0.0,<3.0.0", "3.5.0", "3.5.0"},

		// both ranges
		{"both overlap => keep old", ">=1.0.0,<2.0.0", ">=1.5.0,<1.8.0", ">=1.0.0,<2.0.0"},
		{"both no overlap => pick new", ">=2.0.0,<3.0.0", ">=3.0.0,<4.0.0", ">=3.0.0,<4.0.0"},
		{"both partial overlap => keep old", "~>1.2.0", "^1.5.0", "~>1.2.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldIsVer, oldVer, oldRng, errOld := ParseVersionOrRange(tc.oldInput)
			if errOld != nil {
				t.Fatalf("old parse error: %v", errOld)
			}
			newIsVer, newVer, newRng, errNew := ParseVersionOrRange(tc.newInput)
			if errNew != nil {
				t.Fatalf("new parse error: %v", errNew)
			}
			got := DecideVersionOrRange(
				oldIsVer, oldVer, oldRng, tc.oldInput,
				newIsVer, newVer, newRng, tc.newInput,
			)
			if got != tc.want {
				t.Errorf("DecideVersionOrRange(%q, %q) = %q, want %q",
					tc.oldInput, tc.newInput, got, tc.want)
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
			existingVersion: ">=1.0.0, <3.0.0",
			want:            ">=1.0.0, <3.0.0",
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
			want:            ">=2.0.0, <3.0.0",
		},
		{
			name:            "range: from range",
			strategy:        StrategyRange,
			targetVersion:   ">=2.0.0, <3.0.0",
			existingVersion: ">=1.0.0, <2.0.0",
			want:            ">=2.0.0, <3.0.0",
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
