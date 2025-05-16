package archive

import "strings"

// IsBinaryForPlatform checks if a binary name matches the given OS/architecture
func IsBinaryForPlatform(name, osArch string) bool {
	name = strings.ToLower(name)
	osArch = strings.ToLower(osArch)

	// Split OS and architecture
	parts := strings.Split(osArch, "_")
	if len(parts) != 2 {
		return false
	}
	os, arch := parts[0], parts[1]

	// Special case for Windows executables
	if os == "windows" && strings.HasSuffix(name, ".exe") {
		// For Windows executables, we'll check only the architecture if it's specified
		if strings.Contains(name, arch) {
			return true
		}
		// If no architecture is specified in the name, assume it's compatible
		for _, variant := range []string{arch, "x86_64", "x64", "amd64", "x86", "i386", "i686", "32"} {
			if strings.Contains(name, variant) {
				return false
			}
		}
		return true
	}

	// Check if it's a compressed file first
	if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".zip") {
		// For compressed files, we'll check if the name contains OS and arch
		return strings.Contains(name, os) && strings.Contains(name, arch)
	}

	// Map common architecture names
	archMap := map[string][]string{
		"amd64": {"x86_64", "x64", "amd64"},
		"386":   {"x86", "i386", "i686", "32"},
		"arm64": {"aarch64", "arm64"},
		"arm":   {"armv7", "armhf", "arm"},
	}

	// Map common OS names
	osMap := map[string][]string{
		"darwin":  {"darwin", "macos", "osx"},
		"linux":   {"linux", "gnu"},
		"windows": {"windows", "win"},
	}

	// Check if the binary name contains both OS and architecture
	matchesOS := false
	for _, variant := range osMap[os] {
		if strings.Contains(name, variant) {
			matchesOS = true
			break
		}
	}

	matchesArch := false
	for _, variant := range archMap[arch] {
		if strings.Contains(name, variant) {
			matchesArch = true
			break
		}
	}

	return matchesOS && matchesArch
}
