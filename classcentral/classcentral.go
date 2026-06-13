// Package classcentral is the library behind the cc command: the HTTP client,
// request shaping, and typed data models for Class Central.
//
// Class Central is a course aggregator that embeds Next.js JSON in every page
// under a <script id="__NEXT_DATA__"> tag. The client fetches pages with a
// real browser User-Agent, extracts that JSON, and falls back to HTML card
// parsing when the JSON is absent or shaped differently.
//
// The site is behind Cloudflare. Requests from datacenter IPs receive a
// JS-challenge page instead of content. The client detects the challenge body
// and returns ErrBlocked so commands can report the situation clearly.
package classcentral

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultUserAgent is a real browser UA. It is the single most effective thing
// that keeps polite requests unblocked on sites that check it.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Config holds constructor parameters for Client.
type Config struct {
	// BaseURL is the Class Central root. Default: "https://www.classcentral.com".
	// Tests set this to their httptest.Server URL.
	BaseURL   string
	UserAgent string
	Rate      time.Duration // minimum gap between requests; 0 disables pacing
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible production defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.classcentral.com",
		UserAgent: DefaultUserAgent,
		Rate:      time.Second,
		Retries:   2,
		Timeout:   30 * time.Second,
	}
}

// Client talks to Class Central over HTTP.
type Client struct {
	http      *http.Client
	userAgent string
	baseURL   string
	rate      time.Duration
	retries   int

	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		http:      &http.Client{Timeout: cfg.Timeout},
		userAgent: cfg.UserAgent,
		baseURL:   cfg.BaseURL,
		rate:      cfg.Rate,
		retries:   cfg.Retries,
	}
}

// Get fetches rawURL, paces, retries on transient errors, detects Cloudflare
// challenges, and returns the raw response body.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
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
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	// Class Central returns 403 with a cf-mitigated: challenge header when
	// Cloudflare blocks the request from datacenter IPs.
	if resp.StatusCode == http.StatusForbidden {
		if resp.Header.Get("cf-mitigated") != "" || resp.Header.Get("server") == "cloudflare" {
			return nil, false, ErrBlocked
		}
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}

	if isCloudflare(b) {
		return nil, false, ErrBlocked
	}

	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
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

// ─── public API methods ───────────────────────────────────────────────────────

// Search returns courses matching query. It fetches pages starting at startPage
// (1-based) until limit results are collected or a page returns no results.
func (c *Client) Search(ctx context.Context, query string, startPage, limit int) ([]Course, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []Course
	for page := startPage; ; page++ {
		u := fmt.Sprintf("%s/search?q=%s&page=%d", c.baseURL, encodeQuery(query), page)
		body, err := c.Get(ctx, u)
		if err != nil {
			return out, err
		}
		batch := parseCourses(body, c.baseURL)
		out = append(out, batch...)
		if len(batch) == 0 || len(out) >= limit {
			break
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

// Top returns courses from the top-free-online-courses collection.
func (c *Client) Top(ctx context.Context, limit int) ([]Course, error) {
	u := c.baseURL + "/collection/top-free-online-courses"
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	courses := parseCourses(body, c.baseURL)
	if limit > 0 && limit < len(courses) {
		courses = courses[:limit]
	}
	for i := range courses {
		courses[i].Rank = i + 1
	}
	return courses, nil
}

// Subjects returns all subjects from /subjects.
func (c *Client) Subjects(ctx context.Context) ([]Subject, error) {
	u := c.baseURL + "/subjects"
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	subjects := parseSubjects(body, c.baseURL)
	for i := range subjects {
		subjects[i].Rank = i + 1
	}
	return subjects, nil
}

// Providers returns all providers from /providers.
func (c *Client) Providers(ctx context.Context) ([]Provider, error) {
	u := c.baseURL + "/providers"
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	providers := parseProviders(body, c.baseURL)
	for i := range providers {
		providers[i].Rank = i + 1
	}
	return providers, nil
}

// encodeQuery does minimal URL-encoding for the query parameter.
func encodeQuery(q string) string {
	out := make([]byte, 0, len(q))
	for i := 0; i < len(q); i++ {
		c := q[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~':
			out = append(out, c)
		case c == ' ':
			out = append(out, '+')
		default:
			out = append(out, fmt.Sprintf("%%%02X", c)...)
		}
	}
	return string(out)
}
