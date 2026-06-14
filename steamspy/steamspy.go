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
		BaseURL:   "https://steamspy.com/api.php",
		UserAgent: "steamspy-cli/0.1.0 (github.com/tamnd/steamspy-cli)",
		Rate:      200 * time.Millisecond,
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
func (c *Client) App(ctx context.Context, appID int) (*Game, error) {
	u := fmt.Sprintf("%s?request=appdetails&appid=%d", c.cfg.BaseURL, appID)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var g Game
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, fmt.Errorf("decode app details: %w", err)
	}
	return &g, nil
}

// Genre fetches games in the given genre (paginated, page starts at 0).
func (c *Client) Genre(ctx context.Context, genre string, page int) ([]Game, error) {
	u := fmt.Sprintf("%s?request=genre&genre=%s&page=%d", c.cfg.BaseURL, url.QueryEscape(genre), page)
	return c.fetchGameMap(ctx, u)
}

// Tag fetches games with the given tag (paginated, page starts at 0).
func (c *Client) Tag(ctx context.Context, tag string, page int) ([]Game, error) {
	u := fmt.Sprintf("%s?request=tag&tag=%s&page=%d", c.cfg.BaseURL, url.QueryEscape(tag), page)
	return c.fetchGameMap(ctx, u)
}

// Top fetches the top 100 games. period must be "2weeks", "forever", or "owned".
func (c *Client) Top(ctx context.Context, period string) ([]Game, error) {
	req, err := topRequest(period)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s?request=%s", c.cfg.BaseURL, req)
	return c.fetchGameMap(ctx, u)
}

// topRequest maps the user-facing period name to the API request parameter.
func topRequest(period string) (string, error) {
	switch period {
	case "2weeks", "":
		return "top100in2weeks", nil
	case "forever":
		return "top100forever", nil
	case "owned":
		return "top100owned", nil
	default:
		return "", fmt.Errorf("unknown period %q: must be 2weeks, forever, or owned", period)
	}
}

// fetchGameMap decodes a map[string]Game response and returns games sorted by AppID.
func (c *Client) fetchGameMap(ctx context.Context, u string) ([]Game, error) {
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw map[string]Game
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode game map: %w", err)
	}
	games := make([]Game, 0, len(raw))
	for _, g := range raw {
		games = append(games, g)
	}
	sort.Slice(games, func(i, j int) bool {
		return games[i].AppID < games[j].AppID
	})
	return games, nil
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
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
