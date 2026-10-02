package ai

import (
	"net"
	"net/http"
	"time"
)

// defaultClient is used by a provider that has no HTTPClient set.
var defaultClient = NewHTTPClient(time.Minute)

// NewHTTPClient returns a client for calls to a model provider. Each phase of a
// request has its own deadline. There is no deadline on the whole request: a long
// answer can stream for minutes, and http.Client.Timeout would cut it off.
//
// firstByte is how long to wait for the response headers. For a streamed answer,
// that is the time to first token: model load (cold start) plus prefill.
func NewHTTPClient(firstByte time.Duration) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone() // keeps proxy, HTTP/2 and pool defaults
	t.DialContext = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.TLSHandshakeTimeout = 10 * time.Second
	t.ResponseHeaderTimeout = firstByte
	return &http.Client{Transport: t}
}

// clientOrDefault returns c, or defaultClient when c is nil.
func clientOrDefault(c *http.Client) *http.Client {
	if c == nil {
		return defaultClient
	}
	return c
}
