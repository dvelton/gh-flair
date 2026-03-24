package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/dvelton/gh-flair/internal/model"
)

// UserFetcher looks up GitHub user profiles and caches results in memory.
type UserFetcher struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]*model.NotableUser
}

// NewUserFetcher creates a UserFetcher.
func NewUserFetcher() *UserFetcher {
	return &UserFetcher{
		client: newHTTPClient(),
		cache:  make(map[string]*model.NotableUser),
	}
}

// FetchUser retrieves profile information for a GitHub login.
// Results are cached for the lifetime of the fetcher instance.
func (f *UserFetcher) FetchUser(ctx context.Context, token, login string) (*model.NotableUser, error) {
	f.mu.Lock()
	if cached, ok := f.cache[login]; ok {
		f.mu.Unlock()
		return cached, nil
	}
	f.mu.Unlock()

	url := fmt.Sprintf("https://api.github.com/users/%s", login)
	resp, err := doGet(ctx, f.client, url, ghHeaders(token))
	if err != nil {
		return nil, err
	}
	checkRateLimit(resp)
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return nil, fmt.Errorf("users API %s: %s — %s", url, resp.Status, body)
	}

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		Followers int    `json:"followers"`
		Company   string `json:"company"`
		Location  string `json:"location"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse user %s: %w", login, err)
	}

	user := &model.NotableUser{
		Login:     raw.Login,
		Name:      raw.Name,
		Followers: raw.Followers,
		Company:   raw.Company,
		Location:  raw.Location,
	}

	f.mu.Lock()
	f.cache[login] = user
	f.mu.Unlock()

	return user, nil
}
