package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dvelton/gh-flair/internal/config"
	"github.com/dvelton/gh-flair/internal/presenter"
	"github.com/dvelton/gh-flair/internal/store"
	"github.com/spf13/cobra"
)

var (
	shareFlagRepo      string
	shareFlagMilestone string
	shareFlagOutput    string
	shareFlagFormat    string
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Generate a shareable milestone card.",
	RunE:  runShare,
}

func init() {
	shareCmd.Flags().StringVar(&shareFlagRepo, "repo", "", "repo full name (owner/name) [required]")
	shareCmd.Flags().StringVar(&shareFlagMilestone, "milestone", "", "milestone text to display [required]")
	shareCmd.Flags().StringVar(&shareFlagOutput, "output", "", "output file path (default: auto-generated)")
	shareCmd.Flags().StringVar(&shareFlagFormat, "format", "text", "output format: text or svg")
	_ = shareCmd.MarkFlagRequired("repo")
	_ = shareCmd.MarkFlagRequired("milestone")
}

func runShare(cmd *cobra.Command, args []string) error {
	if shareFlagFormat == "svg" {
		stars := lookupStarCount(shareFlagRepo)

		svg, err := presenter.GenerateShareSVG(shareFlagRepo, shareFlagMilestone, stars)
		if err != nil {
			return fmt.Errorf("generate SVG: %w", err)
		}

		outPath := shareFlagOutput
		if outPath == "" {
			outPath = fmt.Sprintf("%s-milestone-%d.svg", sanitizeFilename(shareFlagRepo), time.Now().Unix())
		}
		if err := os.WriteFile(outPath, svg, 0o644); err != nil {
			return fmt.Errorf("write SVG: %w", err)
		}
		fmt.Printf("✓ SVG written to %s\n", outPath)
		return nil
	}

	fmt.Println(presenter.RenderShareCard(shareFlagRepo, shareFlagMilestone))
	return nil
}

// lookupStarCount fetches the current star count from the store's latest snapshot.
func lookupStarCount(repoFullName string) int {
	st, err := store.New(config.DBPath())
	if err != nil {
		return 0
	}
	defer st.Close()

	repo, err := st.GetRepo(repoFullName)
	if err != nil || repo == nil {
		return 0
	}
	snap, err := st.GetLatestSnapshot(repo.ID)
	if err != nil || snap == nil {
		return 0
	}
	return snap.Stars
}

// sanitizeFilename replaces path separators so the repo name is safe to use as a filename.
func sanitizeFilename(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		if s[i] == '/' || s[i] == '\\' {
			out[i] = '-'
		} else {
			out[i] = s[i]
		}
	}
	return string(out)
}
