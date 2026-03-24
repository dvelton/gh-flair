package analyzer

import (
	"strconv"

	"github.com/dvelton/gh-flair/internal/model"
)

// FilterNotableStargazers takes star events and returns a new slice of
// EventNotableStargazer events where the stargazer's follower count (stored in
// event.Meta["followers"]) meets or exceeds threshold.
func FilterNotableStargazers(events []model.Event, threshold int) []model.Event {
	var out []model.Event

	for _, e := range events {
		if e.Kind != model.EventStar {
			continue
		}

		followers, err := followerCount(e)
		if err != nil || followers < threshold {
			continue
		}

		notable := e
		notable.Kind = model.EventNotableStargazer
		out = append(out, notable)
	}

	return out
}

// followerCount extracts the follower count from event.Meta["followers"].
func followerCount(e model.Event) (int, error) {
	if e.Meta == nil {
		return 0, strconv.ErrSyntax
	}
	raw, ok := e.Meta["followers"]
	if !ok {
		return 0, strconv.ErrSyntax
	}
	return strconv.Atoi(raw)
}
