package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// NpmFetcher retrieves weekly download counts from the npm registry.
type NpmFetcher struct {
	client *http.Client
}

// NewNpmFetcher creates a NpmFetcher.
func NewNpmFetcher() *NpmFetcher {
	return &NpmFetcher{client: newHTTPClient()}
}

// Fetch returns EventDownloadSpike events for any repo that has an "npm" package entry.
func (f *NpmFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	for _, repo := range repos {
		pkg, ok := repo.Packages["npm"]
		if !ok || pkg == "" {
			continue
		}
		evt, err := f.fetchPackage(ctx, repo, pkg)
		if err != nil {
			// Non-fatal: return empty rather than crashing the run.
			continue
		}
		if evt != nil {
			events = append(events, *evt)
		}
	}
	return events, nil
}

type npmDownloadResponse struct {
	Downloads int    `json:"downloads"`
	Package   string `json:"package"`
}

func (f *NpmFetcher) fetchPackage(ctx context.Context, repo model.Repo, pkg string) (*model.Event, error) {
	url := fmt.Sprintf("https://api.npmjs.org/downloads/point/last-week/%s", pkg)
	resp, err := doGet(ctx, f.client, url, map[string]string{"User-Agent": userAgent})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return nil, fmt.Errorf("npm API %s: %s — %s", url, resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var result npmDownloadResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse npm response: %w", err)
	}

	evt := model.Event{
		ID:        fmt.Sprintf("npm-%s-%d", pkg, time.Now().Unix()),
		RepoID:    repo.ID,
		Kind:      model.EventDownloadSpike,
		Title:     fmt.Sprintf("%s downloaded %d times last week (npm)", pkg, result.Downloads),
		Actor:     pkg,
		URL:       fmt.Sprintf("https://www.npmjs.com/package/%s", pkg),
		OccuredAt: time.Now(),
		CreatedAt: time.Now(),
		Meta: map[string]string{
			"registry":  "npm",
			"package":   pkg,
			"downloads": fmt.Sprintf("%d", result.Downloads),
			"period":    "last_week",
		},
	}
	return &evt, nil
}
