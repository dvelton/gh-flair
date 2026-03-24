package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// HNFetcher retrieves Hacker News mentions via the Algolia search API.
type HNFetcher struct {
	client *http.Client
}

// NewHNFetcher creates an HNFetcher.
func NewHNFetcher() *HNFetcher {
	return &HNFetcher{client: newHTTPClient()}
}

// Fetch returns EventHNMention events for each repo that appear after since.
func (f *HNFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	for _, repo := range repos {
		evts, err := f.fetchRepo(ctx, repo, since)
		if err != nil {
			// Non-fatal.
			continue
		}
		events = append(events, evts...)
	}
	return events, nil
}

type hnSearchResponse struct {
	Hits []hnHit `json:"hits"`
}

type hnHit struct {
	ObjectID    string    `json:"objectID"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Points      int       `json:"points"`
	NumComments int       `json:"num_comments"`
	CreatedAt   time.Time `json:"created_at"`
	StoryURL    string    `json:"story_url"`
}

func (f *HNFetcher) fetchRepo(ctx context.Context, repo model.Repo, since time.Time) ([]model.Event, error) {
	query := fmt.Sprintf("github.com/%s/%s", repo.Owner, repo.Name)
	apiURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?query=%s&restrictSearchableAttributes=url&tags=story",
		url.QueryEscape(query),
	)

	resp, err := doGet(ctx, f.client, apiURL, map[string]string{"User-Agent": userAgent})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return nil, fmt.Errorf("HN Algolia API %s: %s — %s", apiURL, resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var result hnSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse HN response: %w", err)
	}

	var events []model.Event
	for _, hit := range result.Hits {
		if hit.CreatedAt.Before(since) {
			continue
		}
		hitURL := hit.URL
		if hitURL == "" {
			hitURL = hit.StoryURL
		}
		events = append(events, model.Event{
			ID:        fmt.Sprintf("hn-%s-%s", repo.FullName, hit.ObjectID),
			RepoID:    repo.ID,
			Kind:      model.EventHNMention,
			Title:     hit.Title,
			Actor:     "",
			URL:       fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID),
			OccuredAt: hit.CreatedAt,
			CreatedAt: time.Now(),
			Meta: map[string]string{
				"points":       fmt.Sprintf("%d", hit.Points),
				"num_comments": fmt.Sprintf("%d", hit.NumComments),
				"title":        hit.Title,
				"story_url":    hitURL,
				"hn_id":        hit.ObjectID,
			},
		})
	}
	return events, nil
}
