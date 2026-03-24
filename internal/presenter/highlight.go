package presenter

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dvelton/gh-flair/internal/model"
)

func renderHeader(since time.Time) string {
	rel := relativeTime(since)
	return styleHeader.Render("✦ gh flair") + styleDim.Render(" — since "+rel)
}

func renderStars(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if s.StarsDelta <= 0 && len(s.NotableStargazers) == 0 {
			continue
		}
		line := fmt.Sprintf("  %s  +%d stars",
			styleBold.Render(s.Repo.FullName),
			s.StarsDelta,
		)
		line += styleDim.Render(fmt.Sprintf(" (%d total)", s.StarsTotal))

		// Proximity hint: if within 5% of a milestone threshold
		if next := nextStarMilestone(s.StarsTotal); next > 0 {
			away := next - s.StarsTotal
			if away > 0 && away <= max(10, next/20) {
				line += "  " + styleDim.Render(fmt.Sprintf("%d away from %d!", away, next))
			}
		}
		lines = append(lines, line)

		for _, ng := range s.NotableStargazers {
			lines = append(lines, fmt.Sprintf("    🐋  %s", styleGreen.Render(ng.Actor)))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return styleGold.Render("★ Stars") + "\n" + strings.Join(lines, "\n")
}

func renderMilestones(milestones []model.MilestoneCelebration) string {
	if len(milestones) == 0 {
		return ""
	}
	var lines []string
	for _, mc := range milestones {
		lines = append(lines, "  "+RenderMilestoneCard(mc))
	}
	return styleHeader.Render("🎉 Milestones") + "\n" + strings.Join(lines, "\n")
}

func renderContributors(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if len(s.NewContributors) == 0 {
			continue
		}
		for _, c := range s.NewContributors {
			actor := c.Actor
			if actor == "" {
				actor = "unknown"
			}
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styleDim.Render(s.Repo.FullName),
				styleGreen.Render(actor),
			))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return styleGreen.Render("👋 New Contributors") + "\n" + strings.Join(lines, "\n")
}

func renderKindWords(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if len(s.GratitudeComments) == 0 {
			continue
		}
		for _, c := range s.GratitudeComments {
			body := truncate(c.Body, 120)
			if body == "" {
				body = c.Title
			}
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styleDim.Render(s.Repo.FullName),
				styleDim.Render(`"`+body+`"`),
			))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return styleCyan.Render("💬 Kind Words") + "\n" + strings.Join(lines, "\n")
}

func renderSponsors(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if len(s.SponsorEvents) == 0 {
			continue
		}
		for _, sp := range s.SponsorEvents {
			actor := sp.Actor
			if actor == "" {
				actor = "anonymous"
			}
			lines = append(lines, fmt.Sprintf("  %s  %s",
				styleDim.Render(s.Repo.FullName),
				stylePurple.Render(actor),
			))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return stylePurple.Render("💰 Sponsors") + "\n" + strings.Join(lines, "\n")
}

func renderDownloads(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if s.DownloadCount <= 0 {
			continue
		}
		delta := ""
		if s.DownloadDelta != 0 {
			sign := "+"
			if s.DownloadDelta < 0 {
				sign = ""
			}
			delta = styleDim.Render(fmt.Sprintf(" (%s%.1f%%)", sign, s.DownloadDelta))
		}
		registry := ""
		if s.DownloadRegistry != "" {
			registry = styleDim.Render(" via "+s.DownloadRegistry)
		}
		lines = append(lines, fmt.Sprintf("  %s  %s%s%s",
			styleDim.Render(s.Repo.FullName),
			styleBlue.Render(fmt.Sprintf("%d downloads", s.DownloadCount)),
			delta,
			registry,
		))
	}
	if len(lines) == 0 {
		return ""
	}
	return styleBlue.Render("📦 Downloads") + "\n" + strings.Join(lines, "\n")
}

func renderHN(summaries []model.RepoSummary) string {
	var lines []string
	for _, s := range summaries {
		if len(s.HNMentions) == 0 {
			continue
		}
		for _, hn := range s.HNMentions {
			points := hn.Meta["points"]
			comments := hn.Meta["comments"]
			meta := ""
			if points != "" || comments != "" {
				meta = styleDim.Render(fmt.Sprintf(" (%s pts, %s comments)", points, comments))
			}
			title := hn.Title
			if title == "" {
				title = s.Repo.FullName
			}
			lines = append(lines, fmt.Sprintf("  %s%s",
				styleCyan.Render(title),
				meta,
			))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return styleCyan.Render("📰 Hacker News") + "\n" + strings.Join(lines, "\n")
}

func renderStreaks(streaks []model.Streak) string {
	var lines []string
	for _, st := range streaks {
		best := ""
		if st.BestDays > st.CurrentDays {
			best = styleDim.Render(fmt.Sprintf(" (best: %d)", st.BestDays))
		}
		lines = append(lines, fmt.Sprintf("  %s  %s%s",
			styleDim.Render(st.Metric),
			styleOrange.Render(fmt.Sprintf("🔥 %d-day streak", st.CurrentDays)),
			best,
		))
	}
	if len(lines) == 0 {
		return ""
	}
	return styleOrange.Render("🔥 Streaks") + "\n" + strings.Join(lines, "\n")
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		h := int(d.Hours())
		// Format like "yesterday 6:42 AM" if > 12h
		if d >= 12*time.Hour {
			return "yesterday " + t.Format("3:04 PM")
		}
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 48*time.Hour:
		return "yesterday " + t.Format("3:04 PM")
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// nextStarMilestone returns the next milestone threshold above current, or 0.
func nextStarMilestone(current int) int {
	for _, t := range model.StarThresholds {
		if t > current {
			return t
		}
	}
	return 0
}

// truncate shortens s to n runes, appending "…" if truncated.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n-1]) + "…"
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
