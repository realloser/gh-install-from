/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/spf13/cobra"
)

var (
	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7B2CBF")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4361EE"))

	cellStyle = lipgloss.NewStyle().
			PaddingRight(2)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4CAF50"))

	repoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF7F50"))

	hostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6"))

	noMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true)
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed binaries",
	Long: `List all binaries installed via gh-install-from.
Shows binary name, version, repository, and GitHub host if available.`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := github.NewGhCliClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	manager, err := binary.New(client)
	if err != nil {
		return fmt.Errorf("failed to create binary manager: %w", err)
	}

	binaries, err := manager.ListInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed binaries: %w", err)
	}

	if len(binaries) == 0 {
		fmt.Println(noMetaStyle.Render("No binaries installed"))
		return nil
	}

	// Print title
	fmt.Println(titleStyle.Render("Installed Binaries"))

	// Print headers
	headers := []string{"Name", "Version", "Repository", "Host"}
	headerRow := make([]string, len(headers))
	for i, header := range headers {
		headerRow[i] = headerStyle.Render(header)
	}
	fmt.Println(strings.Join(headerRow, " "))

	// Print separator
	fmt.Println(strings.Repeat("-", 80))

	// Print binaries
	for _, bin := range binaries {
		name := cellStyle.Render(bin.Name)
		version := cellStyle.Render("--")
		if bin.Version != "" {
			version = cellStyle.Render(versionStyle.Render(bin.Version))
		}
		repo := cellStyle.Render("--")
		if bin.Repository != "" {
			repo = cellStyle.Render(repoStyle.Render(bin.Repository))
		}
		host := "--"
		if bin.Host != "" {
			host = hostStyle.Render(bin.Host)
		}

		fmt.Printf("%s %s %s %s\n",
			name,
			version,
			repo,
			host,
		)
	}

	return nil
}
