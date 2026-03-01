/*
Copyright © 2025 Martyn Messerli
*/
package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/realloser/gh-install-from/pkg/binary"
	"github.com/realloser/gh-install-from/pkg/log"
	"github.com/spf13/cobra"
)

// outdatedCmd lists binaries with available updates
var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List binaries with available updates",
	Long: `List installed binaries that have updates available.
Uses concurrent version checking. Run 'gh install-from update' to update.`,
	RunE: runOutdated,
}

func init() {
	rootCmd.AddCommand(outdatedCmd)
}

func runOutdated(cmd *cobra.Command, args []string) error {
	manager, err := binary.NewManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create binary manager: %w", err)
	}

	binaries, err := manager.ListInstalled()
	if err != nil {
		return fmt.Errorf("failed to list installed: %w", err)
	}

	if len(binaries) == 0 {
		fmt.Println(noMetaStyle.Render("No binaries installed"))
		return nil
	}

	candidates, err := manager.CheckUpdates(binaries)
	if err != nil {
		return fmt.Errorf("failed to check updates: %w", err)
	}

	if len(candidates) == 0 {
		fmt.Println(noMetaStyle.Render("All binaries are up to date"))
		return nil
	}

	log.Debug("outdated binaries", "count", len(candidates))

	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Installed", Width: 15},
		{Title: "Latest", Width: 15},
		{Title: "Repository", Width: 30},
	}

	rows := make([]table.Row, 0, len(candidates))
	for _, c := range candidates {
		rows = append(rows, table.Row{
			c.InstalledBinary.Name,
			c.InstalledBinary.Version,
			c.LatestVersion,
			c.InstalledBinary.Repository,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	t.SetStyles(s)

	fmt.Println(baseStyle.Render(t.View()))
	return nil
}
