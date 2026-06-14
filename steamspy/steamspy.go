// Package steamspy is the library behind the steamspy command line:
// the HTTP client, request shaping, and the typed data models for SteamSpy
// (steamspy.com). No API key required.
//
// The Client sets a real User-Agent, paces requests to stay polite, and
// retries transient failures (429 and 5xx).
package steamspy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "steamspy.com"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://steamspy.com",
		UserAgent: "steamspy-cli/0.1 (tamnd87@gmail.com)",
		Rate:      1 * time.Second,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to steamspy over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// App fetches game details for a given Steam app ID.
func (c *Client) App(ctx context.Context, appID int) (*App, error) {
	u := fmt.Sprintf("%s/api.php?request=appdetails&appid=%d", c.cfg.BaseURL, appID)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var w wireApp
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, fmt.Errorf("decode app details: %w", err)
	}
	a := toApp(w)
	return &a, nil
}

// Top fetches the top 100 games by 2-week players, sorted by Positive desc, limited to limit.
func (c *Client) Top(ctx context.Context, limit int) ([]App, error) {
	u := fmt.Sprintf("%s/api.php?request=top100in2weeks", c.cfg.BaseURL)
	return c.fetchGameMap(ctx, u, limit)
}

// Genre fetches games in the given genre, sorted by Positive desc, limited to limit.
func (c *Client) Genre(ctx context.Context, genre string, limit int) ([]App, error) {
	u := fmt.Sprintf("%s/api.php?request=genre&genre=%s", c.cfg.BaseURL, url.QueryEscape(genre))
	return c.fetchGameMap(ctx, u, limit)
}

// Search searches games by name term, limited to limit results.
func (c *Client) Search(ctx context.Context, term string, limit int) ([]App, error) {
	u := fmt.Sprintf("%s/api.php?request=search&term=%s", c.cfg.BaseURL, url.QueryEscape(term))
	return c.fetchGameMap(ctx, u, limit)
}

// fetchGameMap decodes a map[string]wireApp response, converts to App, sorts by Positive desc, and applies limit.
func (c *Client) fetchGameMap(ctx context.Context, u string, limit int) ([]App, error) {
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	// Some endpoints (notably search) may return an empty body when there are no results.
	if len(body) == 0 {
		return nil, nil
	}
	var raw map[string]wireApp
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode game map: %w", err)
	}
	apps := make([]App, 0, len(raw))
	for _, w := range raw {
		apps = append(apps, toApp(w))
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Positive != apps[j].Positive {
			return apps[i].Positive > apps[j].Positive
		}
		return apps[i].AppID < apps[j].AppID
	})
	if limit > 0 && len(apps) > limit {
		apps = apps[:limit]
	}
	return apps, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}
