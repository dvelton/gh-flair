package fetcher

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

const (
	defaultTimeout   = 10 * time.Second
	rateLimitWarnAt  = 50
	userAgent        = "gh-flair"
)

var rateLimitWarned sync.Once

// Fetcher retrieves events for a set of repos since a given time.
type Fetcher interface {
	Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error)
}

// newHTTPClient returns a client with a default timeout and no automatic redirect following.
func newHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultTimeout}
}

// doGet executes a GET request, setting User-Agent, and optional extra headers.
// It returns the response body bytes. The caller is responsible for checking
// status codes beyond the basic non-200 handling requested by callers.
func doGet(ctx context.Context, client *http.Client, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	return resp, nil
}

// readBody reads and closes a response body.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// checkRateLimit logs a warning (once) when remaining GitHub API calls are low.
func checkRateLimit(resp *http.Response) {
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return
	}
	var n int
	if _, err := fmt.Sscanf(remaining, "%d", &n); err == nil && n < rateLimitWarnAt {
		rateLimitWarned.Do(func() {
			log.Printf("warning: GitHub rate limit low — %d requests remaining", n)
		})
	}
}

// ghHeaders returns the standard headers for GitHub REST API requests.
func ghHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "token " + token,
		"Accept":        "application/vnd.github+json",
		"User-Agent":    userAgent,
	}
}
