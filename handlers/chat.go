package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/ai"
)

// Chat streams an AI response to the browser token-by-token using
// Server-Sent Events (SSE).
//
// The six numbered steps below match the Project 01 slide deck.
func Chat(w http.ResponseWriter, r *http.Request) {
	// 1. Read the message ───────────────────────────────────────────────
	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
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

	// 4. Build the messages slice ───────────────────────────────────────
	// This is the full context the LLM sees: a system prompt + the user's text.
	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant. Answer concisely."},
		{Role: "user", Content: message},
	}

	// 5. Stream tokens to the browser ───────────────────────────────────
	err := ai.ChatStream(ai.DefaultModel, messages, func(token string) error {
		// JSON-encode each token so newlines/quotes can't break the SSE format.
		payload, err := json.Marshal(token)
		if err != nil {
			return err
		}
		// One SSE event = "data: <payload>\n\n"
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return err // browser disconnected — return error to stop the stream
		}
		flusher.Flush() // push this token out immediately, don't buffer
		return nil
	})
	if err != nil {
		// We're already mid-stream, so we can't change the HTTP status now.
		// Log it and signal the frontend with a sentinel it can recognise.
		log.Printf("chat: stream error: %v", err)
		fmt.Fprint(w, "data: \"[ERROR]\"\n\n")
		flusher.Flush()
		return
	}

	// 6. Send the [DONE] sentinel ───────────────────────────────────────
	// The browser watches for this to know the stream is complete.
	fmt.Fprint(w, "data: \"[DONE]\"\n\n")
	flusher.Flush()
}
