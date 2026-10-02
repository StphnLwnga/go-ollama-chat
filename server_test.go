package main

import (
	"net/http"
	"testing"
)

// TestServerTimeouts pins the server's timeout decisions. net/http implements the
// timeouts; this test makes sure they stay configured, and that WriteTimeout stays off.
func TestServerTimeouts(t *testing.T) {
	srv := newServer(":0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout is not set: a slow client can hold a connection open (Slowloris)")
	}
	if srv.ReadTimeout <= 0 {
		t.Error("ReadTimeout is not set: a slow request body can hold a connection open")
	}
	if srv.IdleTimeout <= 0 {
		t.Error("IdleTimeout is not set: idle kept-alive connections are never closed")
	}
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v, want 0: it would cut off every streamed answer", srv.WriteTimeout)
	}
}
