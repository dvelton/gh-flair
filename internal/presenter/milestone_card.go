package presenter

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dvelton/gh-flair/internal/model"
)

// RenderMilestoneCard renders a milestone celebration scaled by its size.
func RenderMilestoneCard(mc model.MilestoneCelebration) string {
	threshold := mc.Milestone.Threshold
	kind := string(mc.Milestone.Kind)

	switch {
	case threshold >= 5000:
		return renderBigMilestoneCard(mc)
	case threshold >= 500:
		return renderMediumMilestoneCard(mc, kind)
	default:
		return renderSmallMilestoneCard(mc, kind)
	}
}

func renderSmallMilestoneCard(mc model.MilestoneCelebration, kind string) string {
	return fmt.Sprintf("%s crossed %s %s",
		styleBold.Render(mc.RepoFullName),
		styleGold.Render(fmt.Sprintf("%d", mc.Milestone.Threshold)),
		kind,
	)
}

func renderMediumMilestoneCard(mc model.MilestoneCelebration, kind string) string {
	line1 := fmt.Sprintf("%s crossed %s %s",
		styleBold.Render(mc.RepoFullName),
		styleGold.Render(fmt.Sprintf("%d", mc.Milestone.Threshold)),
		kind,
	)
	var extras []string
	if mc.PriorMilestone != nil {
		dur := mc.Milestone.CelebratedAt.Sub(mc.PriorMilestone.CelebratedAt).Round(24 * time.Hour)
		days := int(dur.Hours() / 24)
		extras = append(extras, fmt.Sprintf("%d days since %d", days, mc.PriorMilestone.Threshold))
	}
	if mc.Percentile != "" {
		extras = append(extras, mc.Percentile)
	}
	if len(extras) == 0 {
		return line1
	}
	line2 := styleDim.Render(strings.Join(extras, " · "))
	return line1 + "\n  " + line2
}

func renderBigMilestoneCard(mc model.MilestoneCelebration) string {
	kind := string(mc.Milestone.Kind)
	width := 52

	inner := func(s string) string {
		pad := width - 2 - len([]rune(s))
		if pad < 0 {
			pad = 0
		}
		return "┃ " + s + strings.Repeat(" ", pad) + " ┃"
	}

	topBar := "┏" + strings.Repeat("━", width) + "┓"
	botBar := "┗" + strings.Repeat("━", width) + "┛"

	headline := fmt.Sprintf("%s  %s %s",
		mc.RepoFullName,
		fmt.Sprintf("%d", mc.Milestone.Threshold),
		kind,
	)

	boxStyle := lipgloss.NewStyle().
		Foreground(colorGold).
		Bold(true)

	var rows []string
	rows = append(rows, boxStyle.Render(topBar))
	rows = append(rows, boxStyle.Render(inner(headline)))

	if mc.PriorMilestone != nil {
		dur := mc.Milestone.CelebratedAt.Sub(mc.PriorMilestone.CelebratedAt).Round(24 * time.Hour)
		days := int(dur.Hours() / 24)
		detail := fmt.Sprintf("%d days since %d %s", days, mc.PriorMilestone.Threshold, kind)
		rows = append(rows, styleDim.Render(inner(detail)))
	}

	if mc.Percentile != "" {
		rows = append(rows, styleDim.Render(inner(mc.Percentile)))
	}

	rows = append(rows, boxStyle.Render(botBar))

	return strings.Join(rows, "\n")
}
