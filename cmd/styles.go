package cmd

import "github.com/charmbracelet/lipgloss"

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)
	
	// Add more styles as needed
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		PaddingLeft(2)

	subtitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5A5A5A")).
		PaddingLeft(2)
) 