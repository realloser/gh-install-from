/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

type releaseVersion struct {
	tag       string
	createdAt string
}

func (v releaseVersion) Title() string       { return v.tag }
func (v releaseVersion) Description() string { return v.createdAt }
func (v releaseVersion) FilterValue() string { return v.tag }

type model struct {
	list     list.Model
	selected string
	err      error
}

// versionsCmd represents the versions command
var versionsCmd = &cobra.Command{
	Use:   "versions [owner/repo]",
	Short: "List and select available versions",
	Long: `List and select available versions of a binary from a GitHub repository.
Uses an interactive UI to browse and select versions.`,
	Args: cobra.ExactArgs(1),
	RunE: runVersions,
}

func init() {
	rootCmd.AddCommand(versionsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runVersions(cmd *cobra.Command, args []string) error {
	repo := args[0]
	client, err := gh.RESTClient(nil)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		CreatedAt string `json:"created_at"`
	}

	err = client.Get(fmt.Sprintf("repos/%s/releases", repo), &releases)
	if err != nil {
		return fmt.Errorf("failed to get releases: %w", err)
	}

	items := make([]list.Item, len(releases))
	for i, r := range releases {
		items[i] = releaseVersion{
			tag:       r.TagName,
			createdAt: r.CreatedAt,
		}
	}

	m := model{
		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
	m.list.Title = fmt.Sprintf("Available versions for %s", repo)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	if finalModel.(model).selected != "" {
		fmt.Printf("Selected version: %s\n", finalModel.(model).selected)
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(releaseVersion)
			if ok {
				m.selected = i.tag
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	return docStyle.Render(m.list.View())
}
