package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/model"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var streakCmd = &cobra.Command{
	Use:   "streak",
	Short: "Show activity streaks across your repos.",
	RunE:  runStreak,
}

func runStreak(cmd *cobra.Command, args []string) error {
	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	streaks, err := st.GetAllStreaks()
	if err != nil {
		return fmt.Errorf("get streaks: %w", err)
	}
	if len(streaks) == 0 {
		fmt.Println("No streak data yet. Run 'gh flair' to start tracking.")
		return nil
	}

	repos, err := st.ListRepos()
	if err != nil {
		return fmt.Errorf("list repos: %w", err)
	}
	repoByID := make(map[int64]model.Repo, len(repos))
	for _, r := range repos {
		repoByID[r.ID] = r
	}

	// Group streaks by repo ID.
	byRepo := make(map[int64][]model.Streak)
	var repoOrder []int64
	seen := make(map[int64]bool)
	for _, s := range streaks {
		if !seen[s.RepoID] {
			repoOrder = append(repoOrder, s.RepoID)
			seen[s.RepoID] = true
		}
		byRepo[s.RepoID] = append(byRepo[s.RepoID], s)
	}

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	orangeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("🔥 Streaks") + "\n\n")

	for _, repoID := range repoOrder {
		r, ok := repoByID[repoID]
		repoName := fmt.Sprintf("repo #%d", repoID)
		if ok {
			repoName = r.FullName
		}
		sb.WriteString(headerStyle.Render(repoName) + "\n")

		for _, s := range byRepo[repoID] {
			if s.CurrentDays == 0 && s.BestDays == 0 {
				continue
			}
			best := ""
			if s.BestDays > s.CurrentDays {
				best = dimStyle.Render(fmt.Sprintf("  (best: %d)", s.BestDays))
			}
			sb.WriteString(fmt.Sprintf("  %s  %s%s\n",
				dimStyle.Render(s.Metric),
				orangeStyle.Render(fmt.Sprintf("🔥 %d-day streak", s.CurrentDays)),
				best,
			))
		}
		sb.WriteString("\n")
	}

	fmt.Print(sb.String())
	return nil
}
