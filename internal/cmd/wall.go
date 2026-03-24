package cmd

import (
	"fmt"
	"time"

	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/presenter"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var (
	wallFlagSince  string
	wallFlagFormat string
	wallFlagRepo   string
)

var wallCmd = &cobra.Command{
	Use:   "wall",
	Short: "Browse your wall of love.",
	RunE:  runWall,
}

func init() {
	wallCmd.Flags().StringVar(&wallFlagSince, "since", "", "show moments on or after this date (YYYY-MM-DD)")
	wallCmd.Flags().StringVar(&wallFlagFormat, "format", "terminal", "output format: terminal or markdown")
	wallCmd.Flags().StringVar(&wallFlagRepo, "repo", "", "filter by repo full name (owner/name)")
}

func runWall(cmd *cobra.Command, args []string) error {
	st, err := store.New(config.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	filters := store.MomentFilters{
		RepoFullName: wallFlagRepo,
	}
	if wallFlagSince != "" {
		t, err := time.Parse("2006-01-02", wallFlagSince)
		if err != nil {
			return fmt.Errorf("parse --since: expected YYYY-MM-DD, got %q", wallFlagSince)
		}
		filters.Since = t
	}

	moments, err := st.ListMoments(filters)
	if err != nil {
		return fmt.Errorf("list moments: %w", err)
	}

	if len(moments) == 0 {
		fmt.Println("No saved moments yet. Use 'gh flair save <event-id>' to add one.")
		return nil
	}

	if wallFlagFormat == "markdown" {
		fmt.Print(presenter.RenderWallMarkdown(moments))
		return nil
	}

	return presenter.RunWall(moments)
}
