package ai

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeGroq starts a local HTTP server that answers like Groq's streaming API.
func fakeGroq(t *testing.T, status int, lines []string) *httptest.Server {
	t.Helper()
	// http://localhost:11434/api/chat
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-key")
		}
		if status != http.StatusOK {
			http.Error(w, `{"error":{"message":"bad key"}}`, status)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, line := range lines {
			fmt.Fprintf(w, "%s\n\n", line)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestGroqChatStreamSendsTokensInOrder tests the ChatStream method of the Groq struct.
func TestGroqChatStreamSendsTokensInOrder(t *testing.T) {
	srv := fakeGroq(t, http.StatusOK, []string{
		`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: {"choices":[{"delta":{"content":"!"}}]}`,
		`data: [DONE]`,
		`data: {"choices":[{"delta":{"content":"after done"}}]}`,
	})
	g := Groq{APIKey: "test-key", Endpoint: srv.URL}

	var got []string
	err := g.ChatStream("any-model", []Message{{Role: "user", Content: "hi"}}, func(tok string) error {
		got = append(got, tok)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if want := "Hel|lo|!"; strings.Join(got, "|") != want {
		t.Errorf("tokens = %q, want %q", strings.Join(got, "|"), want)
	}
}

func TestGroqChatStreamReturnsErrorOnBadStatus(t *testing.T) {
	srv := fakeGroq(t, http.StatusUnauthorized, nil)
	g := Groq{APIKey: "test-key", Endpoint: srv.URL}

	err := g.ChatStream("any-model", []Message{{Role: "user", Content: "hi"}}, func(string) error { return nil })
	if err == nil {
		t.Fatal("expected an error for a 401 reply, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not mention the 401 status", err)
	}
}

func TestGroqChatStreamStopsWhenCallerStops(t *testing.T) {
	srv := fakeGroq(t, http.StatusOK, []string{
		`data: {"choices":[{"delta":{"content":"one"}}]}`,
		`data: {"choices":[{"delta":{"content":"two"}}]}`,
		`data: [DONE]`,
	})
	g := Groq{APIKey: "test-key", Endpoint: srv.URL}

	calls := 0
	err := g.ChatStream("any-model", []Message{{Role: "user", Content: "hi"}}, func(string) error {
		calls++
		return fmt.Errorf("client went away")
	})
	if err != nil {
		t.Fatalf("stopping early should not be an error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("callback ran %d times, want 1", calls)
	}
}
