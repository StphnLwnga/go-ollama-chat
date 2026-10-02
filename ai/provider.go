package ai

import "context"

// Provider is what a chat backend must offer. handler never see this
// they keep calling the package functions below
type Provider interface {
	Chat(ctx context.Context, model string, messages []Message) (string, error)
	// ChatStream calls onChunk once per token. It stops early, with a non-nil error,
	// when ctx is cancelled or when onChunk fails. An error from onChunk is returned unchanged.
	ChatStream(ctx context.Context, model string, messages []Message, onChunk func(string) error) error
}

// routes is the router built at startup. It is empty until Use is called.
var routes = NewRouter()

// Use installs the router built in main. Call it once, before the server starts.
func Use(r *Router) {
	routes = r
}

// Supports reports whether any provider serves the model.
func Supports(model string) bool {
	return routes.Supports(model)
}

// Models returns the model names the server can serve, sorted.
func Models() []string {
	return routes.Models()
}

// Chat sends the request to the provider for this model.
func Chat(ctx context.Context, model string, messages []Message) (string, error) {
	return routes.Chat(ctx, model, messages)
}

// ChatStream sends the request to the provider for this model.
func ChatStream(ctx context.Context, model string, messages []Message, onChunk func(string) error) error {
	return routes.ChatStream(ctx, model, messages, onChunk)
}
