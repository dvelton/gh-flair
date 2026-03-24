package analyzer

import (
	"strings"

	"github.com/dvelton/gh-flair/internal/model"
)

// positiveKeywords are phrases that indicate genuine appreciation.
var positiveKeywords = []string{
	"thank you",
	"thanks",
	"thank",
	"love this",
	"love it",
	"amazing",
	"awesome",
	"saved me",
	"saved us",
	"great work",
	"appreciate",
	"helpful",
	"well done",
	"fantastic",
	"brilliant",
	"kudos",
	"incredible",
	"life saver",
	"game changer",
}

// negativeKeywords are words that suggest a complaint or problem when found
// near a positive keyword.
var negativeKeywords = []string{
	"bug",
	"broken",
	"issue",
	"unfortunately",
	"however",
	"problem",
	"error",
	"crash",
	"fail",
}

// proximityWindow is the character radius around a positive keyword that is
// checked for negative context.
const proximityWindow = 100

// searchWindow is the maximum offset into the comment body where a positive
// keyword must appear. Keywords in quoted code blocks further along are ignored.
const searchWindow = 500

// FilterGratitude takes a slice of events (all kinds welcome) and returns only
// those EventGratitudeComment events whose body contains a genuine expression
// of gratitude. Matching is case-insensitive.
func FilterGratitude(events []model.Event) []model.Event {
	var out []model.Event

	for _, e := range events {
		if e.Kind != model.EventGratitudeComment {
			continue
		}
		if isGrateful(e.Body) {
			out = append(out, e)
		}
	}

	return out
}

// isGrateful returns true when body contains at least one positive keyword
// within the first searchWindow characters that is not tainted by a negative
// keyword within proximityWindow characters.
func isGrateful(body string) bool {
	lower := strings.ToLower(body)

	// Only consider the first searchWindow characters of the comment body.
	window := lower
	if len(window) > searchWindow {
		window = window[:searchWindow]
	}

	for _, pos := range positiveKeywords {
		idx := strings.Index(window, pos)
		if idx < 0 {
			continue
		}

		// Define the region to check for negative context within the full body.
		start := idx - proximityWindow
		if start < 0 {
			start = 0
		}
		end := idx + len(pos) + proximityWindow
		if end > len(lower) {
			end = len(lower)
		}
		region := lower[start:end]

		tainted := false
		for _, neg := range negativeKeywords {
			if strings.Contains(region, neg) {
				tainted = true
				break
			}
		}

		if !tainted {
			return true
		}
	}

	return false
}
