package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// StarsFetcher retrieves star events from GitHub.
type StarsFetcher struct {
	token  string
	client *http.Client
}

// NewStarsFetcher creates a StarsFetcher with the supplied GitHub token.
func NewStarsFetcher(ghToken string) *StarsFetcher {
	return &StarsFetcher{token: ghToken, client: newHTTPClient()}
}

// Fetch returns EventStar events for each repo since the given time.
func (f *StarsFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
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

type stargazerEntry struct {
	StarredAt time.Time `json:"starred_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type repoInfo struct {
	StargazersCount int `json:"stargazers_count"`
}

func (f *StarsFetcher) fetchRepo(ctx context.Context, repo model.Repo, since time.Time) ([]model.Event, error) {
	totalStars, err := f.fetchStarCount(ctx, repo)
	if err != nil {
		return nil, err
	}

	headers := ghHeaders(f.token)
	headers["Accept"] = "application/vnd.github.star+json"

	var events []model.Event
	page := 1

outer:
	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/stargazers?per_page=100&page=%d&direction=desc",
			repo.Owner, repo.Name, page)

		resp, err := doGet(ctx, f.client, url, headers)
		if err != nil {
			return events, err
		}
		checkRateLimit(resp)
		if resp.StatusCode == http.StatusNotModified {
			break
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := readBody(resp)
			return events, fmt.Errorf("stars API %s: %s — %s", url, resp.Status, body)
		}

		body, err := readBody(resp)
		if err != nil {
			return events, err
		}

		var entries []stargazerEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			return events, fmt.Errorf("parse stargazers: %w", err)
		}
		if len(entries) == 0 {
			break
		}

		for _, e := range entries {
			if e.StarredAt.Before(since) {
				break outer
			}
			events = append(events, model.Event{
				ID:        fmt.Sprintf("star-%s-%s-%d", repo.FullName, e.User.Login, e.StarredAt.Unix()),
				RepoID:    repo.ID,
				Kind:      model.EventStar,
				Title:     fmt.Sprintf("%s starred %s", e.User.Login, repo.FullName),
				Actor:     e.User.Login,
				URL:       fmt.Sprintf("https://github.com/%s", repo.FullName),
				OccuredAt: e.StarredAt,
				CreatedAt: time.Now(),
				Meta: map[string]string{
					"total_stars": strconv.Itoa(totalStars),
				},
			})
		}
		page++
	}

	return events, nil
}

func (f *StarsFetcher) fetchStarCount(ctx context.Context, repo model.Repo) (int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo.Owner, repo.Name)
	resp, err := doGet(ctx, f.client, url, ghHeaders(f.token))
	if err != nil {
		return 0, err
	}
	checkRateLimit(resp)
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return 0, fmt.Errorf("repo API %s: %s — %s", url, resp.Status, body)
	}
	body, err := readBody(resp)
	if err != nil {
		return 0, err
	}
	var info repoInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return 0, fmt.Errorf("parse repo info: %w", err)
	}
	return info.StargazersCount, nil
}
