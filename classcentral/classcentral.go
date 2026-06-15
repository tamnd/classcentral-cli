// Package classcentral is the library behind the cc command: the HTTP client,
// request shaping, and typed data models for Class Central.
//
// Class Central exposes an undocumented but public REST API at
// https://www.classcentral.com/api/. The client sends requests there with a
// real browser User-Agent and parses JSON responses directly. When a response
// is HTML instead of JSON (Cloudflare challenge page), ErrBlocked is returned.
//
// Datacenter IPs may be blocked by Cloudflare. Users on residential IPs or
// running on-device will get live data; CI and datacenter runners will see
// exit 5.
package classcentral

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent is a real browser UA that keeps polite requests unblocked
// on sites that check it.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

// Config holds constructor parameters for Client.
type Config struct {
	// BaseURL is the Class Central API root.
	// Default: "https://www.classcentral.com/api".
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
		BaseURL:   "https://www.classcentral.com/api",
		UserAgent: DefaultUserAgent,
		Rate:      time.Second,
		Retries:   2,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Class Central REST API over HTTP.
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

// Search returns courses matching query. If free is true, only free courses are
// returned. limit controls the maximum number of results.
func (c *Client) Search(ctx context.Context, query string, free bool, limit int) ([]Course, error) {
	if limit <= 0 {
		limit = 20
	}
	const pageSize = 20
	var results []Course
	offset := 0
	for {
		// Request at most what we still need, capped at pageSize.
		need := limit - len(results)
		if need > pageSize {
			need = pageSize
		}
		params := url.Values{
			"limit":  {fmt.Sprint(need)},
			"offset": {fmt.Sprint(offset)},
		}
		if query != "" {
			params.Set("q", query)
		}
		if free {
			params.Set("free", "true")
		}
		var resp searchResp
		if err := c.getJSON(ctx, "/search", params, &resp); err != nil {
			return nil, err
		}
		for i, wc := range resp.Courses {
			results = append(results, wc.toCourse(offset+i+1))
		}
		// Stop when: no results returned, we have enough, or the API signalled
		// there are no more (returned fewer than requested).
		if len(resp.Courses) == 0 || len(results) >= limit || len(resp.Courses) < need {
			break
		}
		offset += need
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// Top returns the top free online courses sorted by enrollment count.
func (c *Client) Top(ctx context.Context, limit int) ([]Course, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{
		"free":  {"true"},
		"sort":  {"student-count"},
		"limit": {fmt.Sprint(limit)},
	}
	var resp searchResp
	if err := c.getJSON(ctx, "/search", params, &resp); err != nil {
		return nil, err
	}
	out := make([]Course, len(resp.Courses))
	for i, wc := range resp.Courses {
		out[i] = wc.toCourse(i + 1)
	}
	return out, nil
}

// Subjects returns all subject categories from the API.
func (c *Client) Subjects(ctx context.Context) ([]Subject, error) {
	params := url.Values{"limit": {"500"}}
	var resp subjectsResp
	if err := c.getJSON(ctx, "/subjects", params, &resp); err != nil {
		return nil, err
	}
	out := make([]Subject, len(resp.Subjects))
	for i, ws := range resp.Subjects {
		out[i] = Subject{
			Rank:  i + 1,
			Name:  ws.Name,
			Count: ws.CourseCount,
			URL:   siteBase(c.baseURL) + "/subject/" + ws.Slug,
		}
	}
	return out, nil
}

// Providers returns all course platforms from the API.
func (c *Client) Providers(ctx context.Context) ([]Provider, error) {
	params := url.Values{"limit": {"500"}}
	var resp providersResp
	if err := c.getJSON(ctx, "/providers", params, &resp); err != nil {
		return nil, err
	}
	out := make([]Provider, len(resp.Providers))
	for i, wp := range resp.Providers {
		out[i] = Provider{
			Rank:    i + 1,
			Name:    wp.Name,
			Courses: wp.CourseCount,
			URL:     siteBase(c.baseURL) + "/provider/" + wp.Slug,
		}
	}
	return out, nil
}

// getJSON fetches path+params from the API base URL, retries on transient
// errors, detects Cloudflare challenge pages, and unmarshals the JSON body
// into out.
func (c *Client) getJSON(ctx context.Context, path string, params url.Values, out any) error {
	rawURL := c.baseURL + path + "?" + params.Encode()
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return json.Unmarshal(body, out)
		}
		lastErr = err
		if !retry {
			return err
		}
	}
	return fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, true, err
	}

	// Detect Cloudflare HTML block regardless of status code.
	if isHTMLBody(b) {
		return nil, false, ErrBlocked
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
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

// isHTMLBody reports whether body is an HTML page (Cloudflare challenge or
// other non-JSON response). Checks the first non-whitespace characters.
func isHTMLBody(b []byte) bool {
	s := strings.TrimSpace(string(b))
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}

// siteBase strips the /api suffix from the base URL to get the site root,
// used when building subject and provider URLs for the human-facing site.
func siteBase(apiBase string) string {
	return strings.TrimSuffix(apiBase, "/api")
}
