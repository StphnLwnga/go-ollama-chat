// Package sse writes Server-Sent Events, with a write deadline on every event.
package sse

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrClientGone marks a failed write or flush: the client closed the connection or stopped reading.
var ErrClientGone = errors.New("client disconnected")

// Writer sends Server-Sent Events. Each event gets its own write deadline,
// so a client that stops reading cannot hold the handler forever.
type Writer struct {
	w       http.ResponseWriter
	rc      *http.ResponseController
	timeout time.Duration
}

// NewWriter returns a Writer that gives each event the given time to be written.
func NewWriter(w http.ResponseWriter, timeout time.Duration) *Writer {
	return &Writer{w: w, rc: http.NewResponseController(w), timeout: timeout}
}

// Send writes data as one JSON-encoded event and flushes it to the client.
// A failed write or flush returns an error that wraps ErrClientGone.
func (s *Writer) Send(data string) error {
	payload, err := json.Marshal(data) // JSON-encode so newlines and quotes cannot break the SSE format
	if err != nil {
		return err
	}
	if err := s.rc.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", payload); err != nil {
		return fmt.Errorf("%w: %w", ErrClientGone, err)
	}
	if err := s.rc.Flush(); err != nil { // unlike http.Flusher, this reports a failed flush
		return fmt.Errorf("%w: %w", ErrClientGone, err)
	}
	return nil
}
