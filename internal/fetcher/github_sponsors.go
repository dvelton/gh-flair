package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
)

const githubGraphQLEndpoint = "https://api.github.com/graphql"

// SponsorsFetcher retrieves sponsor activity via the GitHub GraphQL API.
type SponsorsFetcher struct {
	token  string
	client *http.Client
}

// NewSponsorsFetcher creates a SponsorsFetcher with the supplied GitHub token.
func NewSponsorsFetcher(ghToken string) *SponsorsFetcher {
	return &SponsorsFetcher{token: ghToken, client: newHTTPClient()}
}

// Fetch returns EventSponsor events since the given time.
// repos is accepted for interface compatibility; sponsor activity is viewer-scoped.
func (f *SponsorsFetcher) Fetch(ctx context.Context, repos []model.Repo, since time.Time) ([]model.Event, error) {
	var events []model.Event
	cursor := ""

	for {
		nodes, nextCursor, err := f.fetchPage(ctx, cursor)
		if err != nil {
			return events, err
		}

		done := false
		for _, node := range nodes {
			if node.Timestamp.Before(since) {
				done = true
				break
			}
			if node.Action != "NEW_SPONSORSHIP" && node.Action != "TIER_CHANGE" {
				continue
			}
			events = append(events, model.Event{
				ID:        fmt.Sprintf("sponsor-%s-%d", node.Sponsor.Login, node.Timestamp.Unix()),
				Kind:      model.EventSponsor,
				Title:     fmt.Sprintf("%s became a sponsor", node.Sponsor.Login),
				Actor:     node.Sponsor.Login,
				OccuredAt: node.Timestamp,
				CreatedAt: time.Now(),
				Meta: map[string]string{
					"action":       node.Action,
					"sponsor_name": node.Sponsor.Name,
				},
			})
		}

		if done || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return events, nil
}

type sponsorNode struct {
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Sponsor   struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	} `json:"sponsor"`
}

type sponsorsResponse struct {
	Data struct {
		Viewer struct {
			SponsorsActivities struct {
				Nodes    []sponsorNode `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"sponsorsActivities"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

const sponsorsQuery = `
query($cursor: String) {
  viewer {
    sponsorsActivities(first: 50, after: $cursor, orderBy: {field: TIMESTAMP, direction: DESC}) {
      nodes {
        action
        timestamp
        sponsor {
          ... on User { login name }
          ... on Organization { login name }
        }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

func (f *SponsorsFetcher) fetchPage(ctx context.Context, cursor string) ([]sponsorNode, string, error) {
	variables := map[string]interface{}{"cursor": nil}
	if cursor != "" {
		variables["cursor"] = cursor
	}

	payload := map[string]interface{}{
		"query":     sponsorsQuery,
		"variables": variables,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("marshal graphql payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQLEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("build graphql request: %w", err)
	}
	req.Header.Set("Authorization", "token "+f.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("graphql request: %w", err)
	}
	checkRateLimit(resp)
	body, err := readBody(resp)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("graphql API: %s — %s", resp.Status, body)
	}

	var result sponsorsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("parse graphql response: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, "", fmt.Errorf("graphql error: %s", result.Errors[0].Message)
	}

	acts := result.Data.Viewer.SponsorsActivities
	nextCursor := ""
	if acts.PageInfo.HasNextPage {
		nextCursor = acts.PageInfo.EndCursor
	}
	return acts.Nodes, nextCursor, nil
}
