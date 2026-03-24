package presenter

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dvelton/gh-flair/internal/model"
)

// kindIcon returns an emoji for each event kind.
func kindIcon(k model.EventKind) string {
	switch k {
	case model.EventStar:
		return "★"
	case model.EventFork:
		return "⑂"
	case model.EventFirstTimePR, model.EventMergedPR:
		return "⬡"
	case model.EventGratitudeComment:
		return "💬"
	case model.EventSponsor:
		return "💰"
	case model.EventNotableStargazer:
		return "🐋"
	case model.EventHNMention:
		return "📰"
	case model.EventDownloadSpike:
		return "📦"
	default:
		return "✦"
	}
}

// wallStyle for the outer container
var wallStyle = lipgloss.NewStyle().Padding(0, 1)

// wallModel is the Bubble Tea model for the wall of love.
type wallModel struct {
	viewport viewport.Model
	moments  []model.SavedMoment
	ready    bool
}

func (m wallModel) Init() tea.Cmd {
	return nil
}

func (m wallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.SetContent(renderMomentsContent(m.moments, msg.Width-4))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m wallModel) View() string {
	if !m.ready {
		return "\n  Loading…"
	}
	help := styleDim.Render("  ↑/↓ or j/k to scroll · q to quit")
	return wallStyle.Render(m.viewport.View()) + "\n" + help
}

// renderMomentsContent builds the full scrollable content for the wall.
func renderMomentsContent(moments []model.SavedMoment, width int) string {
	var sb strings.Builder
	for i, m := range moments {
		if i > 0 {
			sb.WriteString("\n")
		}
		icon := kindIcon(m.Kind)
		header := fmt.Sprintf("%s  %s  %s  %s",
			styleDim.Render(m.SavedAt.Format("2006-01-02")),
			styleGold.Render(icon),
			styleBold.Render(m.Actor),
			styleDim.Render(fmt.Sprintf("repo #%d", m.RepoID)),
		)
		sb.WriteString(header + "\n")

		body := m.Body
		if body == "" {
			body = m.Title
		}
		if body != "" {
			wrapped := wordWrap(body, width-4)
			sb.WriteString(styleDim.Render("  "+wrapped) + "\n")
		}
	}
	return sb.String()
}

// RunWall launches the interactive Bubble Tea wall of love.
func RunWall(moments []model.SavedMoment) error {
	m := wallModel{moments: moments}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RenderWallMarkdown exports the wall as plain markdown for non-interactive use.
func RenderWallMarkdown(moments []model.SavedMoment) string {
	var sb strings.Builder
	sb.WriteString("# Wall of Love\n\n")
	for _, m := range moments {
		icon := kindIcon(m.Kind)
		sb.WriteString(fmt.Sprintf("## %s %s — %s\n\n", icon, m.Actor, m.SavedAt.Format("Jan 2, 2006")))
		body := m.Body
		if body == "" {
			body = m.Title
		}
		if body != "" {
			sb.WriteString(body + "\n\n")
		}
		if m.URL != "" {
			sb.WriteString(fmt.Sprintf("[View](%s)\n\n", m.URL))
		}
		sb.WriteString("---\n\n")
	}
	return sb.String()
}

// wordWrap wraps text at word boundaries to fit within width.
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n  ")
}
