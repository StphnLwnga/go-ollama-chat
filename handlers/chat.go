package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/ai"
)

// Chat streams an AI response to the browser token-by-token using
// Server-Sent Events (SSE), using the whole conversation as context.
func Chat(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the conversation ─────────────────────────────────────────
	// The browser sends the entire conversation as JSON: {"messages":[...]}.
	// Re-sending every prior turn each time is what gives the chat its
	// "memory" — the model itself remembers nothing between requests.
	var req struct {
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

	// 2. Set SSE headers ────────────────────────────────────────────────
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // stops proxies (e.g. Nginx) from buffering the stream

	// 3. Assert http.Flusher ────────────────────────────────────────────
	// Flush() is what physically pushes each token to the browser.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 4. Stream tokens to the browser ───────────────────────────────────
	// Pass the full conversation straight to Ollama; each token comes back
	// via the callback, which we forward to the browser as an SSE event.
	err := ai.ChatStream(ai.DefaultModel, req.Messages, func(token string) error {
		payload, err := json.Marshal(token) // JSON-encode so newlines/quotes can't break the SSE format
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return err // browser disconnected — return error to stop the stream
		}
		flusher.Flush() // push this token out immediately
		return nil
	})
	if err != nil {
		log.Printf("chat: stream error: %v", err)
		fmt.Fprint(w, "data: \"[ERROR]\"\n\n")
		flusher.Flush()
		return
	}

	// 5. Send the [DONE] sentinel ───────────────────────────────────────
	fmt.Fprint(w, "data: \"[DONE]\"\n\n")
	flusher.Flush()
}
