package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// providerCase is one provider under contract test.
type providerCase struct {
	newProvider func(t *testing.T, endpoint string) Provider // builds the provider against a fake server
	twoTokens   string                                       // a streamed reply of two tokens, in the provider's wire format
}

// providerCases lists every provider. Every contract test runs against all of them.
var providerCases = map[string]providerCase{
	"ollama": {
		newProvider: func(t *testing.T, endpoint string) Provider {
			u, err := url.Parse(endpoint)
			if err != nil {
				t.Fatal(err)
			}
			return Ollama{BaseURL: u}
		},
		twoTokens: `{"message":{"role":"assistant","content":"one"},"done":false}` + "\n" +
			`{"message":{"role":"assistant","content":"two"},"done":true}` + "\n",
	},
	"groq": {
		newProvider: func(t *testing.T, endpoint string) Provider {
			return Groq{APIKey: "test-key", Endpoint: endpoint}
		},
		twoTokens: `data: {"choices":[{"delta":{"content":"one"}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{"content":"two"}}]}` + "\n\n" +
			"data: [DONE]\n\n",
	},
}

// TestProvidersStopWhenContextIsCancelled is a contract test: every provider must pass it.
// The fake server holds the request open, like a model in a long prefill,
// until the client goes away.
func TestProvidersStopWhenContextIsCancelled(t *testing.T) {
	methods := map[string]func(ctx context.Context, p Provider) error{
		"Chat": func(ctx context.Context, p Provider) error {
			_, err := p.Chat(ctx, "any-model", nil)
			return err
		},
		"ChatStream": func(ctx context.Context, p Provider) error {
			return p.ChatStream(ctx, "any-model", nil, func(string) error { return nil })
		},
	}

	for pname, pc := range providerCases {
		for mname, call := range methods {
			t.Run(pname+"/"+mname, func(t *testing.T) {
				t.Parallel()
				started := make(chan struct{})
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					io.Copy(io.Discard, r.Body) // the server notices a client disconnect only after the body is read
					close(started)
					select {
					case <-r.Context().Done(): // the client went away
					case <-time.After(2 * time.Second): // upper bound, so a provider that ignores ctx cannot hang the test
					}
				}))
				t.Cleanup(srv.Close)

				ctx, cancel := context.WithCancel(t.Context())
				go func() {
					<-started // cancel only after the request reached the server
					cancel()
				}()

				err := call(ctx, pc.newProvider(t, srv.URL))
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err = %v, want context.Canceled", err)
				}
			})
		}
	}
}

// TestProvidersReturnTheCallbackError is a contract test: when onChunk fails,
// ChatStream stops at once and returns that same error.
func TestProvidersReturnTheCallbackError(t *testing.T) {
	errStop := errors.New("caller stopped")
	for name, pc := range providerCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, pc.twoTokens)
			}))
			t.Cleanup(srv.Close)

			calls := 0
			err := pc.newProvider(t, srv.URL).ChatStream(t.Context(), "any-model", nil, func(string) error {
				calls++
				return errStop
			})
			if !errors.Is(err, errStop) {
				t.Fatalf("err = %v, want the callback's error", err)
			}
			if calls != 1 {
				t.Errorf("callback ran %d times, want 1", calls)
			}
		})
	}
}
