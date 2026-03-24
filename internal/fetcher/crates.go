package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// CratesFetcher retrieves download counts from crates.io.
type CratesFetcher struct {
	client *http.Client
}

// NewCratesFetcher creates a CratesFetcher.
func NewCratesFetcher() *CratesFetcher {
	return &CratesFetcher{client: newHTTPClient()}
}

// Fetch returns EventDownloadSpike events for any repo that has a "crates" package entry.
func (f *CratesFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	for _, repo := range repos {
		pkg, ok := repo.Packages["crates"]
		if !ok || pkg == "" {
			continue
		}
		evt, err := f.fetchCrate(ctx, repo, pkg)
		if err != nil {
			// Non-fatal.
			continue
		}
		if evt != nil {
			events = append(events, *evt)
		}
	}
	return events, nil
}

type crateResponse struct {
	Crate struct {
		RecentDownloads int    `json:"recent_downloads"`
		Name            string `json:"name"`
	} `json:"crate"`
}

func (f *CratesFetcher) fetchCrate(ctx context.Context, repo model.Repo, pkg string) (*model.Event, error) {
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s", pkg)
	// crates.io requires a descriptive User-Agent or it returns 403.
	resp, err := doGet(ctx, f.client, url, map[string]string{
		"User-Agent": userAgent + " (github.com/dvelton/gh-flair)",
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return nil, fmt.Errorf("crates.io API %s: %s — %s", url, resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var result crateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse crates.io response: %w", err)
	}

	evt := model.Event{
		ID:        fmt.Sprintf("crates-%s-%d", pkg, time.Now().Unix()),
		RepoID:    repo.ID,
		Kind:      model.EventDownloadSpike,
		Title:     fmt.Sprintf("%s downloaded %d times recently (crates.io)", pkg, result.Crate.RecentDownloads),
		Actor:     pkg,
		URL:       fmt.Sprintf("https://crates.io/crates/%s", pkg),
		OccuredAt: time.Now(),
		CreatedAt: time.Now(),
		Meta: map[string]string{
			"registry":  "crates",
			"package":   pkg,
			"downloads": fmt.Sprintf("%d", result.Crate.RecentDownloads),
			"period":    "recent",
		},
	}
	return &evt, nil
}
