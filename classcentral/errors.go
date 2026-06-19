package classcentral

import "errors"

// ErrBlocked is returned when the response body is a Cloudflare JS challenge
// or any HTML block page rather than JSON.
var ErrBlocked = errors.New("blocked by Cloudflare")

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")
