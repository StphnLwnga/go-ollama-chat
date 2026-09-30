package ai

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// ErrUnknownModel means no provider serves the requested model.
var ErrUnknownModel = errors.New("ai: unknown model")

// Router is a Provider that sends each request to the provider registered for its model.
type Router struct {
	routes map[string]Provider
}

// Compile-time check: *Router satisfies Provider.
var _ Provider = (*Router)(nil)

// NewRouter returns an empty router. Add models with Register.
func NewRouter() *Router {
	return &Router{routes: make(map[string]Provider)}
}

// Register makes p serve the given models. Call it only at startup.
func (r *Router) Register(p Provider, models ...string) {
	for _, m := range models {
		r.routes[m] = p
	}
}

// Supports reports whether any provider serves the model.
func (r *Router) Supports(model string) bool {
	_, ok := r.routes[model]
	return ok
}

// Models returns the registered model names in sorted order.
func (r *Router) Models() []string {
	return slices.Sorted(maps.Keys(r.routes))
}

// resolve finds the provider registered for model. It returns ErrUnknownModel if none is registered.
func (r *Router) resolve(model string) (Provider, error) {
	p, ok := r.routes[model]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownModel, model)
	}
	return p, nil
}

// Chat sends the request to the provider for this model.
func (r *Router) Chat(model string, messages []Message) (string, error) {
	p, err := r.resolve(model)
	if err != nil {
		return "", err
	}
	return p.Chat(model, messages)
}

// ChatStream sends the request to the provider for this model.
func (r *Router) ChatStream(model string, messages []Message, onChunk func(string) error) error {
	p, err := r.resolve(model)
	if err != nil {
		return err
	}
	return p.ChatStream(model, messages, onChunk)
}
