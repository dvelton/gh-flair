package presenter

import (
	"fmt"
	"strings"
)

// RenderShareCard generates a formatted text block for copy-pasting to social media.
func RenderShareCard(repoName string, milestoneText string) string {
	width := 50
	border := strings.Repeat("─", width)

	lines := []string{
		"┌" + border + "┐",
		padLine("", width),
		padLine("  ✦  "+repoName, width),
		padLine("", width),
		padLine("  "+milestoneText, width),
		padLine("", width),
		"└" + border + "┘",
	}

	card := styleGold.Render(strings.Join(lines, "\n"))
	return card + "\n" + styleDim.Render("built with github.com/dvelton/gh-flair")
}

func padLine(content string, width int) string {
	runes := []rune(content)
	pad := width - len(runes)
	if pad < 0 {
		pad = 0
	}
	return "│" + content + strings.Repeat(" ", pad) + "│"
}

// GenerateShareSVG generates a simple SVG image card suitable for sharing.
func GenerateShareSVG(repoName string, milestoneText string, stars int) ([]byte, error) {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="600" height="200" viewBox="0 0 600 200">
  <rect width="600" height="200" rx="12" fill="#0d1117"/>
  <rect x="1" y="1" width="598" height="198" rx="11" fill="none" stroke="#30363d" stroke-width="1"/>
  <!-- star icon -->
  <text x="32" y="56" font-size="28" fill="#ffd700">★</text>
  <!-- repo name -->
  <text x="68" y="56" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="22" font-weight="700" fill="#f0f6fc">%s</text>
  <!-- milestone text -->
  <text x="32" y="100" font-family="-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif" font-size="18" fill="#8b949e">%s</text>
  <!-- star count badge -->
  <rect x="32" y="130" width="120" height="32" rx="6" fill="#161b22" stroke="#30363d" stroke-width="1"/>
  <text x="52" y="151" font-family="ui-monospace,SFMono-Regular,Menlo,monospace" font-size="14" fill="#ffd700">★ %s stars</text>
  <!-- footer -->
  <text x="32" y="182" font-family="-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif" font-size="12" fill="#484f58">gh flair</text>
</svg>`, repoName, milestoneText, formatStars(stars))

	return []byte(svg), nil
}

func formatStars(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
