// Package semver provides semantic version parsing and comparison functionality
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
}

// Parse parses a version string into a Version object
func Parse(version string) (*Version, error) {
	v := &Version{}

	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split build metadata
	parts := strings.SplitN(version, "+", 2)
	if len(parts) == 2 {
		v.Build = parts[1]
	}
	version = parts[0]

	// Split pre-release
	parts = strings.SplitN(version, "-", 2)
	if len(parts) == 2 {
		v.PreRelease = parts[1]
	}
	version = parts[0]

	// Split version numbers
	parts = strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", version)
	}

	var err error
	if v.Major, err = strconv.Atoi(parts[0]); err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}
	if v.Minor, err = strconv.Atoi(parts[1]); err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}
	if v.Patch, err = strconv.Atoi(parts[2]); err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return v, nil
}

// GT returns true if v is greater than other
func (v *Version) GT(other *Version) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch > other.Patch
	}
	if v.PreRelease == "" && other.PreRelease != "" {
		return true
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return false
	}
	return v.PreRelease > other.PreRelease
}

// String returns the string representation of the version
func (v *Version) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		version += "-" + v.PreRelease
	}
	if v.Build != "" {
		version += "+" + v.Build
	}
	return version
}
