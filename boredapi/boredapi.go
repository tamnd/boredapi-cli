// Package boredapi is the library behind the boredapi command line:
// the HTTP client, request shaping, and typed data models for the Bored API.
//
// The Client sets a real User-Agent, paces requests, and retries transient
// failures (429 and 5xx) with exponential backoff. One operation is provided:
// get a random activity suggestion, with optional filters by type, participants,
// or key.
package boredapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "bored.api.lewagon.com"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// Config holds tunable knobs for the HTTP client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns sensible defaults for production use.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: "boredapi-cli/0.1 (tamnd87@gmail.com)",
		Rate:      200 * time.Millisecond,
		Timeout:   10 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Bored API over HTTP.
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

// wireActivity is the raw JSON shape returned by /api/activity.
type wireActivity struct {
	Activity      string  `json:"activity"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
	Accessibility float64 `json:"accessibility"`
	Error         string  `json:"error"`
}

// Activity is the public output record for the activity operation.
type Activity struct {
	Key           string `kit:"id" json:"key"`
	Activity      string `json:"activity"`
	Type          string `json:"type"`
	Participants  int    `json:"participants"`
	Price         string `json:"price"`
	Accessibility string `json:"accessibility"`
	Link          string `json:"link"`
}

// ActivityFilter holds optional query parameters for GetActivity.
type ActivityFilter struct {
	Type         string
	Participants int
	Key          string
}

// GetActivity returns a random activity, optionally filtered.
func (c *Client) GetActivity(ctx context.Context, f ActivityFilter) (*Activity, error) {
	rawURL := c.cfg.BaseURL + "/api/activity"
	if q := buildQuery(f); q != "" {
		rawURL += "?" + q
	}
	b, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var w wireActivity
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, fmt.Errorf("decode activity: %w", err)
	}
	if w.Error != "" {
		return nil, fmt.Errorf("%s", w.Error)
	}
	return &Activity{
		Key:           w.Key,
		Activity:      w.Activity,
		Type:          w.Type,
		Participants:  w.Participants,
		Price:         fmt.Sprintf("%.2f", w.Price),
		Accessibility: fmt.Sprintf("%.2f", w.Accessibility),
		Link:          w.Link,
	}, nil
}

// buildQuery constructs the URL query string from a filter.
func buildQuery(f ActivityFilter) string {
	v := url.Values{}
	if f.Type != "" {
		v.Set("type", f.Type)
	}
	if f.Participants > 0 {
		v.Set("participants", fmt.Sprintf("%d", f.Participants))
	}
	if f.Key != "" {
		v.Set("key", f.Key)
	}
	return v.Encode()
}

// get fetches rawURL and returns the response body. It paces and retries.
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

// pace blocks until at least Rate has passed since the previous request.
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
		d = 5 * time.Second
	}
	return d
}
