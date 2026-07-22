package utils

import (
	"net/http"
	"time"
)

// HTTPClient is a shared HTTP client for outbound calls to external services
// (Google/Microsoft OAuth token endpoints, calendar APIs, ICS feeds). Go's
// default client has NO timeout, so a slow or hanging upstream would block the
// request — and therefore the page — indefinitely. A bounded timeout turns a
// hang into a normal error, which the callers already handle. 10s is generous
// for these APIs while still capping the worst case.
var HTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}
