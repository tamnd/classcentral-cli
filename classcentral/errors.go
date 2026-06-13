package classcentral

import (
	"bytes"
	"errors"
)

// ErrBlocked is returned when the response body is a Cloudflare JS challenge.
var ErrBlocked = errors.New("blocked by Cloudflare")

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// isCloudflare reports whether body is a Cloudflare challenge page.
// The challenge sets a window._cf_chl_opt global that is unique to Cloudflare.
func isCloudflare(body []byte) bool {
	return bytes.Contains(body, []byte("cf_chl_opt"))
}
