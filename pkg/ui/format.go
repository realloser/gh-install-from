// Package ui provides UI formatting utilities
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Action styles
	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#2ECC71")). // Green
			PaddingRight(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")). // Blue
			PaddingRight(1)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E74C3C")) // Red

	// Content styles
	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BDC3C7")). // Light gray
			Italic(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")) // Purple
)

// FormatBinaryInfo formats binary information with consistent styling
func FormatBinaryInfo(name, path, version string) string {
	var parts []string
	if name != "" {
		parts = append(parts, name)
	}
	if path != "" {
		parts = append(parts, pathStyle.Render("in "+path))
	}
	if version != "" {
		parts = append(parts, versionStyle.Render("("+version+")"))
	}
	return infoStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, parts...))
}

// FormatActionMessage formats an action message with consistent styling
func FormatActionMessage(action, details string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		successStyle.Render(action),
		details,
	)
}

// FormatErrorMessage formats an error message with consistent styling
func FormatErrorMessage(message string) string {
	return errorStyle.Render(message)
}
