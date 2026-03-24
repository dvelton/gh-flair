package presenter

import (
	"fmt"
	"strings"

	"github.com/dvelton/gh-flair/internal/model"
)

// RenderRecap renders a yearly or monthly summary view across all repos.
func RenderRecap(repos []model.RepoSummary, period string) string {
	var lines []string

	lines = append(lines, styleHeader.Render("✦ gh flair — "+period))
	lines = append(lines, "")

	for _, s := range repos {
		if s.StarsDelta == 0 && len(s.NewContributors) == 0 &&
			s.DownloadCount == 0 && len(s.SponsorEvents) == 0 {
			continue
		}

		lines = append(lines, styleBold.Render(s.Repo.FullName))

		if s.StarsDelta > 0 {
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styleGold.Render(fmt.Sprintf("+%d stars", s.StarsDelta)),
				styleDim.Render(fmt.Sprintf("(%d total)", s.StarsTotal)),
			))
		}

		if n := len(s.NewContributors); n > 0 {
			noun := "contributors"
			if n == 1 {
				noun = "contributor"
			}
			lines = append(lines, fmt.Sprintf("  %s",
				styleGreen.Render(fmt.Sprintf("%d new %s", n, noun)),
			))
		}

		if s.DownloadCount > 0 {
			delta := ""
			if s.DownloadDelta != 0 {
				sign := "+"
				if s.DownloadDelta < 0 {
					sign = ""
				}
				delta = styleDim.Render(fmt.Sprintf(" (%s%.1f%%)", sign, s.DownloadDelta))
			}
			lines = append(lines, fmt.Sprintf("  %s%s",
				styleBlue.Render(fmt.Sprintf("%d downloads", s.DownloadCount)),
				delta,
			))
		}

		if n := len(s.SponsorEvents); n > 0 {
			noun := "sponsors"
			if n == 1 {
				noun = "sponsor"
			}
			lines = append(lines, fmt.Sprintf("  %s",
				stylePurple.Render(fmt.Sprintf("%d %s", n, noun)),
			))
		}

		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
