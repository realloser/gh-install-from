// Package config provides configuration for gh-install-from adapters
package config

import (
	"os"
	"runtime"
	"strings"
)

// Config holds adapter selection and other settings
type Config struct {
	Client   string // "gh" or "mock"
	Store    string // "json"
	OS       string // "darwin", "linux", "windows", "unix"
	Shell    string // "bash", "zsh", "fish", "powershell"
	ManagerOS string // "darwin", "linux", "windows"
}

// FromEnv returns Config populated from environment variables
func FromEnv() *Config {
	return &Config{
		Client:     getEnv("GH_INSTALL_FROM_CLIENT", "gh"),
		Store:      getEnv("GH_INSTALL_FROM_STORE", "json"),
		OS:         getEnv("GH_INSTALL_FROM_OS", runtime.GOOS),
		Shell:      getEnv("GH_INSTALL_FROM_SHELL", ""),
		ManagerOS:  getEnv("GH_INSTALL_FROM_MANAGER_OS", runtime.GOOS),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultVal
}
