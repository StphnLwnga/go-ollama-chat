package sse

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSendGivesUpOnAClientThatStopsReading: a client that never reads must not
// hold the handler. Once the socket buffers are full, the write deadline fails Send.
func TestSendGivesUpOnAClientThatStopsReading(t *testing.T) {
	result := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse := NewWriter(w, 100*time.Millisecond)
		big := strings.Repeat("x", 64<<10)
		for {
			if err := sse.Send(big); err != nil {
				result <- err
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })                       // runs before srv.Close, and unblocks a stuck handler
	fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: test\r\n\r\n") // then never read the reply

	select {
	case err := <-result:
		if !errors.Is(err, ErrClientGone) {
			t.Fatalf("err = %v, want ErrClientGone", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send still blocked after 5s: the write has no deadline")
	}
}
