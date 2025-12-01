package deps

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"},
		},
		{
			name:  "version with v prefix",
			input: "v1.2.3",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Raw: "1.2.3"},
		},
		{
			name:  "version with prerelease",
			input: "1.2.3-beta.1",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta.1", Raw: "1.2.3-beta.1"},
		},
		{
			name:  "version with build metadata",
			input: "1.2.3+build.123",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Build: "build.123", Raw: "1.2.3+build.123"},
		},
		{
			name:  "version with prerelease and build",
			input: "1.2.3-alpha+build",
			want:  &Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha", Build: "build", Raw: "1.2.3-alpha+build"},
		},
		{
			name:  "major only",
			input: "1",
			want:  &Version{Major: 1, Minor: 0, Patch: 0, Raw: "1"},
		},
		{
			name:  "major.minor only",
			input: "1.2",
			want:  &Version{Major: 1, Minor: 2, Patch: 0, Raw: "1.2"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid major",
			input:   "a.2.3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
			if got.Prerelease != tt.want.Prerelease {
				t.Errorf("ParseVersion() prerelease = %v, want %v", got.Prerelease, tt.want.Prerelease)
			}
			if got.Build != tt.want.Build {
				t.Errorf("ParseVersion() build = %v, want %v", got.Build, tt.want.Build)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("ParseVersion() raw = %v, want %v", got.Raw, tt.want.Raw)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{name: "equal", v1: "1.2.3", v2: "1.2.3", want: 0},
		{name: "major less", v1: "1.2.3", v2: "2.2.3", want: -1},
		{name: "major greater", v1: "2.2.3", v2: "1.2.3", want: 1},
		{name: "minor less", v1: "1.2.3", v2: "1.3.3", want: -1},
		{name: "minor greater", v1: "1.3.3", v2: "1.2.3", want: 1},
		{name: "patch less", v1: "1.2.3", v2: "1.2.4", want: -1},
		{name: "patch greater", v1: "1.2.4", v2: "1.2.3", want: 1},
		{name: "prerelease less than release", v1: "1.2.3-alpha", v2: "1.2.3", want: -1},
		{name: "release greater than prerelease", v1: "1.2.3", v2: "1.2.3-alpha", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, _ := ParseVersion(tt.v1)
			v2, _ := ParseVersion(tt.v2)
			if got := v1.Compare(v2); got != tt.want {
				t.Errorf("Version.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseConstraint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOp  string
		wantVer string
		wantErr bool
	}{
		{name: "exact", input: "=1.2.3", wantOp: "=", wantVer: "1.2.3"},
		{name: "gte", input: ">=1.2.3", wantOp: ">=", wantVer: "1.2.3"},
		{name: "gt", input: ">1.2.3", wantOp: ">", wantVer: "1.2.3"},
		{name: "lte", input: "<=1.2.3", wantOp: "<=", wantVer: "1.2.3"},
		{name: "lt", input: "<1.2.3", wantOp: "<", wantVer: "1.2.3"},
		{name: "tilde", input: "~1.2.3", wantOp: "~", wantVer: "1.2.3"},
		{name: "caret", input: "^1.2.3", wantOp: "^", wantVer: "1.2.3"},
		{name: "implicit exact", input: "1.2.3", wantOp: "=", wantVer: "1.2.3"},
		{name: "wildcard", input: "*", wantOp: "*", wantVer: ""},
		{name: "empty", input: "", wantOp: "*", wantVer: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConstraint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConstraint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Op != tt.wantOp {
				t.Errorf("ParseConstraint() op = %v, want %v", got.Op, tt.wantOp)
			}
			if tt.wantVer != "" && got.Version.Raw != tt.wantVer {
				t.Errorf("ParseConstraint() version = %v, want %v", got.Version.Raw, tt.wantVer)
			}
		})
	}
}

func TestConstraintMatches(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		{name: "exact match", constraint: "=1.2.3", version: "1.2.3", want: true},
		{name: "exact no match", constraint: "=1.2.3", version: "1.2.4", want: false},
		{name: "gte match equal", constraint: ">=1.2.3", version: "1.2.3", want: true},
		{name: "gte match greater", constraint: ">=1.2.3", version: "1.2.4", want: true},
		{name: "gte no match", constraint: ">=1.2.3", version: "1.2.2", want: false},
		{name: "gt match", constraint: ">1.2.3", version: "1.2.4", want: true},
		{name: "gt no match equal", constraint: ">1.2.3", version: "1.2.3", want: false},
		{name: "lte match equal", constraint: "<=1.2.3", version: "1.2.3", want: true},
		{name: "lte match less", constraint: "<=1.2.3", version: "1.2.2", want: true},
		{name: "lte no match", constraint: "<=1.2.3", version: "1.2.4", want: false},
		{name: "lt match", constraint: "<1.2.3", version: "1.2.2", want: true},
		{name: "lt no match equal", constraint: "<1.2.3", version: "1.2.3", want: false},
		{name: "tilde match patch", constraint: "~1.2.3", version: "1.2.5", want: true},
		{name: "tilde no match minor", constraint: "~1.2.3", version: "1.3.0", want: false},
		{name: "caret match minor", constraint: "^1.2.3", version: "1.5.0", want: true},
		{name: "caret no match major", constraint: "^1.2.3", version: "2.0.0", want: false},
		{name: "wildcard", constraint: "*", version: "1.2.3", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := ParseConstraint(tt.constraint)
			v, _ := ParseVersion(tt.version)
			if got := c.Matches(v); got != tt.want {
				t.Errorf("Constraint.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraints(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		version string
		want    bool
	}{
		{
			name:    "single constraint",
			input:   ">=1.2.0",
			version: "1.3.0",
			want:    true,
		},
		{
			name:    "range constraint",
			input:   ">=1.2.0 <2.0.0",
			version: "1.5.0",
			want:    true,
		},
		{
			name:    "range constraint lower bound",
			input:   ">=1.2.0 <2.0.0",
			version: "1.2.0",
			want:    true,
		},
		{
			name:    "range constraint upper bound fail",
			input:   ">=1.2.0 <2.0.0",
			version: "2.0.0",
			want:    false,
		},
		{
			name:    "range constraint below lower",
			input:   ">=1.2.0 <2.0.0",
			version: "1.1.0",
			want:    false,
		},
		{
			name:    "wildcard",
			input:   "*",
			version: "5.0.0",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc, err := ParseVersionConstraints(tt.input)
			if err != nil {
				t.Fatalf("ParseVersionConstraints() error = %v", err)
			}
			if got := vc.MatchesString(tt.version); got != tt.want {
				t.Errorf("VersionConstraints.MatchesString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		name string
		c1   string
		c2   string
		want bool
	}{
		{
			name: "compatible ranges",
			c1:   ">=1.2.0",
			c2:   "<2.0.0",
			want: true,
		},
		{
			name: "incompatible ranges",
			c1:   ">=2.0.0",
			c2:   "<1.5.0",
			want: false,
		},
		{
			name: "exact with compatible range",
			c1:   "=1.5.0",
			c2:   ">=1.0.0 <2.0.0",
			want: true,
		},
		{
			name: "exact with incompatible range",
			c1:   "=2.5.0",
			c2:   ">=1.0.0 <2.0.0",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc1, _ := ParseVersionConstraints(tt.c1)
			vc2, _ := ParseVersionConstraints(tt.c2)
			if got := IsCompatible(vc1, vc2); got != tt.want {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}
