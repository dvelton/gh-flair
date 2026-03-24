package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// EventsFetcher retrieves fork and first-time contributor PR events from GitHub.
type EventsFetcher struct {
	token  string
	client *http.Client
}

// NewEventsFetcher creates an EventsFetcher with the supplied GitHub token.
func NewEventsFetcher(ghToken string) *EventsFetcher {
	return &EventsFetcher{token: ghToken, client: newHTTPClient()}
}

// Fetch returns EventFork and EventFirstTimePR events for each repo since the given time.
func (f *EventsFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	for _, repo := range repos {
		evts, err := f.fetchRepo(ctx, repo, since)
		if err != nil {
			return events, err
		}
		events = append(events, evts...)
	}
	return events, nil
}

type ghEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Actor     ghActor         `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
}

type ghActor struct {
	Login string `json:"login"`
}

type prPayload struct {
	Action      string `json:"action"`
	PullRequest struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		HTMLURL   string `json:"html_url"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
}

func (f *EventsFetcher) fetchRepo(ctx context.Context, repo model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	page := 1

outer:
	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/events?per_page=100&page=%d",
			repo.Owner, repo.Name, page)

		resp, err := doGet(ctx, f.client, url, ghHeaders(f.token))
		if err != nil {
			return events, err
		}
		checkRateLimit(resp)
		if resp.StatusCode == http.StatusNotModified {
			break
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := readBody(resp)
			return events, fmt.Errorf("events API %s: %s — %s", url, resp.Status, body)
		}

		body, err := readBody(resp)
		if err != nil {
			return events, err
		}

		var ghEvents []ghEvent
		if err := json.Unmarshal(body, &ghEvents); err != nil {
			return events, fmt.Errorf("parse events: %w", err)
		}
		if len(ghEvents) == 0 {
			break
		}

		for _, e := range ghEvents {
			if e.CreatedAt.Before(since) {
				break outer
			}
			switch e.Type {
			case "ForkEvent":
				events = append(events, model.Event{
					ID:        fmt.Sprintf("fork-%s-%s", repo.FullName, e.ID),
					RepoID:    repo.ID,
					Kind:      model.EventFork,
					Title:     fmt.Sprintf("%s forked %s", e.Actor.Login, repo.FullName),
					Actor:     e.Actor.Login,
					URL:       fmt.Sprintf("https://github.com/%s/%s", e.Actor.Login, repo.Name),
					OccuredAt: e.CreatedAt,
					CreatedAt: time.Now(),
					Meta:      map[string]string{},
				})

			case "PullRequestEvent":
				var pr prPayload
				if err := json.Unmarshal(e.Payload, &pr); err != nil {
					continue
				}
				if pr.Action != "opened" {
					continue
				}
				isFirst, err := f.isFirstTimeContributor(ctx, repo, pr.PullRequest.User.Login)
				if err != nil || !isFirst {
					continue
				}
				events = append(events, model.Event{
					ID:        fmt.Sprintf("first-pr-%s-%d", repo.FullName, pr.PullRequest.Number),
					RepoID:    repo.ID,
					Kind:      model.EventFirstTimePR,
					Title:     fmt.Sprintf("First contribution from %s: %s", pr.PullRequest.User.Login, pr.PullRequest.Title),
					Body:      pr.PullRequest.Title,
					Actor:     pr.PullRequest.User.Login,
					URL:       pr.PullRequest.HTMLURL,
					OccuredAt: e.CreatedAt,
					CreatedAt: time.Now(),
					Meta: map[string]string{
						"pr_number": fmt.Sprintf("%d", pr.PullRequest.Number),
					},
				})
			}
		}
		page++
	}

	return events, nil
}

// isFirstTimeContributor returns true when the login has no prior merged PRs to this repo.
func (f *EventsFetcher) isFirstTimeContributor(ctx context.Context, repo model.Repo, login string) (bool, error) {
	url := fmt.Sprintf(
		"https://api.github.com/search/issues?q=repo:%s+author:%s+type:pr+is:merged&per_page=1",
		repo.FullName, login,
	)

	headers := ghHeaders(f.token)
	headers["Accept"] = "application/vnd.github+json"

	resp, err := doGet(ctx, f.client, url, headers)
	if err != nil {
		return false, err
	}
	checkRateLimit(resp)
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return false, fmt.Errorf("search API: %s — %s", resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return false, err
	}

	var result struct {
		TotalCount int `json:"total_count"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("parse search result: %w", err)
	}

	return result.TotalCount == 0, nil
}
