package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// PyPIFetcher retrieves weekly download counts from PyPI Stats.
type PyPIFetcher struct {
	client *http.Client
}

// NewPyPIFetcher creates a PyPIFetcher.
func NewPyPIFetcher() *PyPIFetcher {
	return &PyPIFetcher{client: newHTTPClient()}
}

// Fetch returns EventDownloadSpike events for any repo that has a "pypi" package entry.
func (f *PyPIFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	for _, repo := range repos {
		pkg, ok := repo.Packages["pypi"]
		if !ok || pkg == "" {
			continue
		}
		evt, err := f.fetchPackage(ctx, repo, pkg)
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

type pypiRecentResponse struct {
	Data struct {
		LastWeek  int `json:"last_week"`
		LastMonth int `json:"last_month"`
		LastDay   int `json:"last_day"`
	} `json:"data"`
	Package string `json:"package"`
}

func (f *PyPIFetcher) fetchPackage(ctx context.Context, repo model.Repo, pkg string) (*model.Event, error) {
	url := fmt.Sprintf("https://pypistats.org/api/packages/%s/recent", pkg)
	resp, err := doGet(ctx, f.client, url, map[string]string{"User-Agent": userAgent})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return nil, fmt.Errorf("pypistats API %s: %s — %s", url, resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var result pypiRecentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse pypistats response: %w", err)
	}

	evt := model.Event{
		ID:        fmt.Sprintf("pypi-%s-%d", pkg, time.Now().Unix()),
		RepoID:    repo.ID,
		Kind:      model.EventDownloadSpike,
		Title:     fmt.Sprintf("%s downloaded %d times last week (PyPI)", pkg, result.Data.LastWeek),
		Actor:     pkg,
		URL:       fmt.Sprintf("https://pypi.org/project/%s/", pkg),
		OccuredAt: time.Now(),
		CreatedAt: time.Now(),
		Meta: map[string]string{
			"registry":  "pypi",
			"package":   pkg,
			"downloads": fmt.Sprintf("%d", result.Data.LastWeek),
			"period":    "last_week",
		},
	}
	return &evt, nil
}
