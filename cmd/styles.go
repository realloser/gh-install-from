package cmd

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Define styles for consistent UI
	docStyle = lipgloss.NewStyle().Margin(1, 2)
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
