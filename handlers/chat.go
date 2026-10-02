package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/StphnLwnga/go-ollama-chat/ai"
	"github.com/StphnLwnga/go-ollama-chat/db"
	"github.com/StphnLwnga/go-ollama-chat/sse"
)

// eventWriteTimeout bounds each event write. A browser that is reading takes
// microseconds; only a client that has stopped reading reaches this limit.
const eventWriteTimeout = 10 * time.Second

// Chat streams an AI response to the browser token-by-token using
// Server-Sent Events (SSE), using the whole conversation as context.
func Chat(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the conversation ─────────────────────────────────────────
	// The browser sends the entire conversation as JSON: {"messages":[...]}.
	// Re-sending every prior turn each time is what gives the chat its
	// "memory" — the model itself remembers nothing between requests.
	var req struct {
		Model    string       `json:"model"` // chosen at runtime by the browser (E5)
		Messages []ai.Message `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "messages is required", http.StatusBadRequest)
		return
	}

	// Fall back to the default if the client didn't send a model.
	model := req.Model // required: the allowlist check below rejects an empty or unknown model

	// Reject models the server cannot route, before any side effect or upstream call.
	if !ai.Supports(model) {
		http.Error(w, fmt.Sprintf("unknown model %q", model), http.StatusBadRequest)
		return
	}

	// 2. Set SSE headers ────────────────────────────────────────────────
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // stops proxies (e.g. Nginx) from buffering the stream

	// 3. Check that the response can stream ────────────────────────────
	// Flushing is what physically pushes each token to the browser.
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	events := sse.NewWriter(w, eventWriteTimeout)

	// 4. Persist the new user turn (the last message in the conversation) ─
	last := req.Messages[len(req.Messages)-1]
	if last.Role == "user" {
		if err := db.Save(last.Role, last.Content); err != nil {
			log.Printf("chat: save user message: %v", err)
		}
	}

	// 5. Stream tokens to the browser, building up the full reply as we go ─
	// Pass the full conversation to Ollama; each token comes back via the
	// callback, which we forward to the browser AND append to `reply`.
	var reply strings.Builder
	err := ai.ChatStream(r.Context(), model, req.Messages, func(token string) error {
		reply.WriteString(token)  // accumulate so we can save the whole reply
		return events.Send(token) // an error stops the stream
	})
	if r.Context().Err() != nil || errors.Is(err, sse.ErrClientGone) {
		// The client left. That is not a server error, and nobody is there to read [ERROR].
		// The reply is cut off, so it is not saved as a complete answer.
		log.Printf("chat: client left after %d bytes of reply", reply.Len())
		return
	}
	if err != nil {
		log.Printf("chat: stream error: %v", err)
		events.Send("[ERROR]")
		return
	}

	// 6. Persist the assistant's reply now that it's complete ────────────
	if reply.Len() > 0 {
		if err := db.Save("assistant", reply.String()); err != nil {
			log.Printf("chat: save assistant message: %v", err)
		}
	}

	// 7. Send the [DONE] sentinel ───────────────────────────────────────
	events.Send("[DONE]")
}
