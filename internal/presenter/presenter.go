package presenter

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dvelton/gh-flair/internal/model"
)

// Color palette
var (
	colorGold    = lipgloss.Color("220")
	colorGreen   = lipgloss.Color("82")
	colorBlue    = lipgloss.Color("75")
	colorPurple  = lipgloss.Color("213")
	colorOrange  = lipgloss.Color("208")
	colorCyan    = lipgloss.Color("87")
	colorDim     = lipgloss.Color("242")
	colorWhite   = lipgloss.Color("255")
)

// Styles
var (
	styleGold   = lipgloss.NewStyle().Foreground(colorGold)
	styleGreen  = lipgloss.NewStyle().Foreground(colorGreen)
	styleBlue   = lipgloss.NewStyle().Foreground(colorBlue)
	stylePurple = lipgloss.NewStyle().Foreground(colorPurple)
	styleOrange = lipgloss.NewStyle().Foreground(colorOrange)
	styleCyan   = lipgloss.NewStyle().Foreground(colorCyan)
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleHeader = lipgloss.NewStyle().Foreground(colorWhite).Bold(true)
	styleBold   = lipgloss.NewStyle().Bold(true)
)

// RenderHighlightReel assembles all sections into a single terminal output string.
func RenderHighlightReel(reel *model.HighlightReel) string {
	var sections []string

	header := renderHeader(reel.Since)
	if header != "" {
		sections = append(sections, header)
	}

	if s := renderStars(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderMilestones(reel.Milestones); s != "" {
		sections = append(sections, s)
	}
	if s := renderContributors(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderKindWords(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderSponsors(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderDownloads(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderHN(reel.RepoSummaries); s != "" {
		sections = append(sections, s)
	}
	if s := renderStreaks(reel.Streaks); s != "" {
		sections = append(sections, s)
	}

	return strings.Join(sections, "\n\n")
}
