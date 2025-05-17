/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/github"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/spf13/cobra"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

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

	log.Debug("creating table for binaries", "count", len(binaries))

	// Create table columns
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Version", Width: 15},
		{Title: "Repository", Width: 30},
		{Title: "Host", Width: 20},
	}

	// Create table rows
	rows := make([]table.Row, 0, len(binaries))
	for _, bin := range binaries {
		name := bin.Name
		if bin.OriginalBinary != "" && bin.OriginalBinary != bin.Name {
			name = fmt.Sprintf("%s (%s)", bin.Name, bin.OriginalBinary)
		}
		version := bin.Version
		if version == "" {
			version = "--"
		}
		repo := bin.Repository
		if repo == "" {
			repo = "--"
		}
		host := bin.Host
		if host == "" {
			host = "--"
		}
		log.Debug("adding row to table", "name", name, "version", version, "repo", repo, "host", host)
		rows = append(rows, table.Row{name, version, repo, host})
	}

	log.Debug("created table rows", "count", len(rows))

	// Create and style the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1), // Add 1 for header row
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	t.SetStyles(s)

	log.Debug("created table", "height", len(rows)+1, "focused", false)

	// Render the table directly
	fmt.Println(baseStyle.Render(t.View()))
	return nil
}
