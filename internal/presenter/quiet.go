package presenter

import (
	"fmt"
	"strings"

	"github.com/dvelton/gh-flair/internal/model"
)

// RenderQuiet renders a compact one-liner suitable for a shell prompt.
func RenderQuiet(reel *model.HighlightReel) string {
	var parts []string

	totalStars := 0
	for _, s := range reel.RepoSummaries {
		totalStars += s.StarsDelta
	}
	if totalStars > 0 {
		parts = append(parts, styleGold.Render(fmt.Sprintf("+%d stars", totalStars)))
	}

	totalContributors := 0
	for _, s := range reel.RepoSummaries {
		totalContributors += len(s.NewContributors)
	}
	if totalContributors > 0 {
		noun := "new contributors"
		if totalContributors == 1 {
			noun = "new contributor"
		}
		parts = append(parts, styleGreen.Render(fmt.Sprintf("%d %s", totalContributors, noun)))
	}

	totalComments := 0
	for _, s := range reel.RepoSummaries {
		totalComments += len(s.GratitudeComments)
	}
	if totalComments > 0 {
		noun := "kind comments"
		if totalComments == 1 {
			noun = "kind comment"
		}
		parts = append(parts, styleCyan.Render(fmt.Sprintf("%d %s", totalComments, noun)))
	}

	totalSponsors := 0
	for _, s := range reel.RepoSummaries {
		totalSponsors += len(s.SponsorEvents)
	}
	if totalSponsors > 0 {
		noun := "new sponsors"
		if totalSponsors == 1 {
			noun = "new sponsor"
		}
		parts = append(parts, stylePurple.Render(fmt.Sprintf("%d %s", totalSponsors, noun)))
	}

	for _, st := range reel.Streaks {
		if st.CurrentDays > 1 {
			parts = append(parts, styleOrange.Render(fmt.Sprintf("🔥 %d-day streak", st.CurrentDays)))
			break
		}
	}

	if len(parts) == 0 {
		return styleDim.Render("✦ nothing new")
	}
	return styleDim.Render("✦ ") + strings.Join(parts, styleDim.Render(" · "))
}
