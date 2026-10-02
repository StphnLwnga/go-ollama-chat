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

// TestProvidersStopWhenContextIsCancelled is a contract test: every provider must pass it.
// The fake server holds the request open, like a model in a long prefill,
// until the client goes away.
func TestProvidersStopWhenContextIsCancelled(t *testing.T) {
	providers := map[string]func(t *testing.T, endpoint string) Provider{
		"ollama": func(t *testing.T, endpoint string) Provider {
			u, err := url.Parse(endpoint)
			if err != nil {
				t.Fatal(err)
			}
			return Ollama{BaseURL: u}
		},
		"groq": func(t *testing.T, endpoint string) Provider {
			return Groq{APIKey: "test-key", Endpoint: endpoint}
		},
	}
	methods := map[string]func(ctx context.Context, p Provider) error{
		"Chat": func(ctx context.Context, p Provider) error {
			_, err := p.Chat(ctx, "any-model", nil)
			return err
		},
		"ChatStream": func(ctx context.Context, p Provider) error {
			return p.ChatStream(ctx, "any-model", nil, func(string) error { return nil })
		},
	}

	for pname, newProvider := range providers {
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

				err := call(ctx, newProvider(t, srv.URL))
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("err = %v, want context.Canceled", err)
				}
			})
		}
	}
}
