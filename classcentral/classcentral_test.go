package classcentral

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetReturnsErrBlockedOnCloudflare(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal Cloudflare challenge body
		_, _ = w.Write([]byte(`<html><body><script>window._cf_chl_opt={cType:"managed"};</script></body></html>`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get(context.Background(), srv.URL)
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("want ErrBlocked, got %v", err)
	}
}

func TestGetStopsRetryingOn403(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.retries = 5

	_, err := c.Get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if hits != 1 {
		t.Errorf("server saw %d hits, want 1 (no retry on 403)", hits)
	}
}
