package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/db"
)

// History returns every saved message as JSON so the browser can rebuild the
// conversation after a refresh.
func History(w http.ResponseWriter, r *http.Request) {
	msgs, err := db.Load()
	if err != nil {
		http.Error(w, "could not load history", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(msgs)
}
