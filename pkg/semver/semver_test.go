package semver

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    *Version
		wantErr bool
	}{
		{
			name:    "basic version",
			version: "1.2.3",
			want:    &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "version with v prefix",
			version: "v1.2.3",
			want:    &Version{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "version with pre-release",
			version: "1.2.3-alpha.1",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha.1"},
		},
		{
			name:    "version with v prefix and pre-release",
			version: "v1.2.3-alpha.1",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha.1"},
		},
		{
			name:    "version with build metadata",
			version: "1.2.3+20130313144700",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, Build: "20130313144700"},
		},
		{
			name:    "version with v prefix and build metadata",
			version: "v1.2.3+20130313144700",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, Build: "20130313144700"},
		},
		{
			name:    "version with pre-release and build metadata",
			version: "1.2.3-beta.1+20130313144700",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta.1", Build: "20130313144700"},
		},
		{
			name:    "version with v prefix, pre-release and build metadata",
			version: "v1.2.3-beta.1+20130313144700",
			want:    &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta.1", Build: "20130313144700"},
		},
		{
			name:    "invalid version format",
			version: "1.2",
			wantErr: true,
		},
		{
			name:    "invalid major version",
			version: "a.2.3",
			wantErr: true,
		},
		{
			name:    "invalid minor version",
			version: "1.b.3",
			wantErr: true,
		},
		{
			name:    "invalid patch version",
			version: "1.2.c",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got.Major != tt.want.Major ||
				got.Minor != tt.want.Minor ||
				got.Patch != tt.want.Patch ||
				got.PreRelease != tt.want.PreRelease ||
				got.Build != tt.want.Build {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_GT(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		other   *Version
		want    bool
	}{
		{
			name:    "greater major version",
			version: &Version{Major: 2, Minor: 0, Patch: 0},
			other:   &Version{Major: 1, Minor: 9, Patch: 9},
			want:    true,
		},
		{
			name:    "greater minor version",
			version: &Version{Major: 1, Minor: 2, Patch: 0},
			other:   &Version{Major: 1, Minor: 1, Patch: 9},
			want:    true,
		},
		{
			name:    "greater patch version",
			version: &Version{Major: 1, Minor: 1, Patch: 2},
			other:   &Version{Major: 1, Minor: 1, Patch: 1},
			want:    true,
		},
		{
			name:    "equal versions",
			version: &Version{Major: 1, Minor: 1, Patch: 1},
			other:   &Version{Major: 1, Minor: 1, Patch: 1},
			want:    false,
		},
		{
			name:    "pre-release less than release",
			version: &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"},
			other:   &Version{Major: 1, Minor: 0, Patch: 0},
			want:    false,
		},
		{
			name:    "release greater than pre-release",
			version: &Version{Major: 1, Minor: 0, Patch: 0},
			other:   &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.GT(tt.other); got != tt.want {
				t.Errorf("Version.GT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		want    string
	}{
		{
			name:    "basic version",
			version: &Version{Major: 1, Minor: 2, Patch: 3},
			want:    "1.2.3",
		},
		{
			name:    "version with pre-release",
			version: &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha.1"},
			want:    "1.2.3-alpha.1",
		},
		{
			name:    "version with build metadata",
			version: &Version{Major: 1, Minor: 2, Patch: 3, Build: "20130313144700"},
			want:    "1.2.3+20130313144700",
		},
		{
			name:    "version with pre-release and build metadata",
			version: &Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta.1", Build: "20130313144700"},
			want:    "1.2.3-beta.1+20130313144700",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
