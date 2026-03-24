package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

// CommentsFetcher retrieves issue and PR comments from GitHub.
type CommentsFetcher struct {
	token  string
	client *http.Client
}

// NewCommentsFetcher creates a CommentsFetcher with the supplied GitHub token.
func NewCommentsFetcher(ghToken string) *CommentsFetcher {
	return &CommentsFetcher{token: ghToken, client: newHTTPClient()}
}

// Fetch returns comment events for each repo since the given time.
// Sentiment filtering is left to the analyzer layer; this fetcher returns all comments.
func (f *CommentsFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
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

type issueComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	IssueURL string `json:"issue_url"`
}

func (f *CommentsFetcher) fetchRepo(ctx context.Context, repo model.Repo, since time.Time) ([]model.Event, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	baseURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues/comments?since=%s&sort=created&direction=desc&per_page=100",
		repo.Owner, repo.Name, url.QueryEscape(sinceStr),
	)

	var events []model.Event
	page := 1

	for {
		pageURL := baseURL + "&page=" + strconv.Itoa(page)
		resp, err := doGet(ctx, f.client, pageURL, ghHeaders(f.token))
		if err != nil {
			return events, err
		}
		checkRateLimit(resp)
		if resp.StatusCode == http.StatusNotModified {
			break
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := readBody(resp)
			return events, fmt.Errorf("comments API %s: %s — %s", pageURL, resp.Status, body)
		}

		body, err := readBody(resp)
		if err != nil {
			return events, err
		}

		var comments []issueComment
		if err := json.Unmarshal(body, &comments); err != nil {
			return events, fmt.Errorf("parse comments: %w", err)
		}
		if len(comments) == 0 {
			break
		}

		for _, c := range comments {
			if c.CreatedAt.Before(since) {
				continue
			}
			issueNum := extractIssueNumber(c.IssueURL)
			events = append(events, model.Event{
				ID:        fmt.Sprintf("comment-%s-%d", repo.FullName, c.ID),
				RepoID:    repo.ID,
				Kind:      model.EventGratitudeComment,
				Title:     fmt.Sprintf("Comment by %s on %s#%s", c.User.Login, repo.FullName, issueNum),
				Body:      c.Body,
				Actor:     c.User.Login,
				URL:       c.HTMLURL,
				OccuredAt: c.CreatedAt,
				CreatedAt: time.Now(),
				Meta: map[string]string{
					"issue_number": issueNum,
					"author":       c.User.Login,
				},
			})
		}
		page++
	}

	return events, nil
}

// extractIssueNumber pulls the issue/PR number from a GitHub issue_url like
// https://api.github.com/repos/owner/repo/issues/42
func extractIssueNumber(issueURL string) string {
	if issueURL == "" {
		return ""
	}
	parsed, err := url.Parse(issueURL)
	if err != nil {
		return ""
	}
	segments := parsed.Path
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i] == '/' {
			return segments[i+1:]
		}
	}
	return ""
}
