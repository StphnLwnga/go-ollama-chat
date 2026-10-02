package ai

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// spyProvider records which models and which context it received, and returns a fixed reply.
type spyProvider struct {
	name  string
	calls []string
	ctx   context.Context
}

func (s *spyProvider) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	s.calls = append(s.calls, model)
	s.ctx = ctx
	return s.name, nil
}

func (s *spyProvider) ChatStream(ctx context.Context, model string, messages []Message, onChunk func(string) error) error {
	s.calls = append(s.calls, model)
	s.ctx = ctx
	return onChunk(s.name)
}

// TestRouterSendsEachModelToItsProvider: tests the router sends each request to the provider for the given model.
func TestRouterSendsEachModelToItsProvider(t *testing.T) {
	local := &spyProvider{name: "local"}
	hosted := &spyProvider{name: "hosted"}

	r := NewRouter()
	r.Register(local, "small")
	r.Register(hosted, "big-a", "big-b")

	var got string
	err := r.ChatStream(t.Context(), "big-b", nil, func(chunk string) error {
		got = chunk
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if got != "hosted" {
		t.Errorf("reply came from %q, want %q", got, "hosted")
	}
	if !slices.Equal(hosted.calls, []string{"big-b"}) {
		t.Errorf("hosted calls = %v, want [big-b]", hosted.calls)
	}
	if len(local.calls) != 0 {
		t.Errorf("local provider was called: %v", local.calls)
	}
}

// TestRouterRejectsUnknownModel: tests the router rejects unknown models.
func TestRouterRejectsUnknownModel(t *testing.T) {
	local := &spyProvider{name: "local"}
	r := NewRouter()
	r.Register(local, "small")

	_, err := r.Chat(t.Context(), "gpt-4", nil)
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
	if r.Supports("gpt-4") {
		t.Error("Supports(gpt-4) = true, want false")
	}
	if len(local.calls) != 0 {
		t.Errorf("a provider was called for an unknown model: %v", local.calls)
	}
}

// TestRouterModelsAreSorted: tests the Models() method returns the registered model names in sorted order.
func TestRouterModelsAreSorted(t *testing.T) {
	r := NewRouter()
	r.Register(&spyProvider{}, "zeta", "alpha")
	r.Register(&spyProvider{}, "mid")

	want := []string{"alpha", "mid", "zeta"}
	if got := r.Models(); !slices.Equal(got, want) {
		t.Errorf("Models() = %v, want %v", got, want)
	}
}

// ctxKey is a private key type. No other package can make the same key, so no
// other package can read or overwrite the value stored under it.
type ctxKey struct{}

// TestRouterPassesContextToProvider: the router gives the provider the caller's context, not a new one.
func TestRouterPassesContextToProvider(t *testing.T) {
	spy := &spyProvider{name: "hosted"}
	r := NewRouter()
	r.Register(spy, "big")

	calls := map[string]func(ctx context.Context) error{
		"Chat": func(ctx context.Context) error {
			_, err := r.Chat(ctx, "big", nil)
			return err
		},
		"ChatStream": func(ctx context.Context) error {
			return r.ChatStream(ctx, "big", nil, func(string) error { return nil })
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(t.Context(), ctxKey{}, name)
			if err := call(ctx); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if spy.ctx == nil {
				t.Fatal("provider got no context")
			}
			if got := spy.ctx.Value(ctxKey{}); got != name {
				t.Errorf("provider context value = %v, want %q", got, name)
			}
		})
	}
}
