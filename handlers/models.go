package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/ai"
)

// Models returns the list of available AI models as JSON.
func Models(w http.ResponseWriter, r *http.Request) {
	models := ai.Models()
	if models == nil {
		models = []string{} // encode an empty list rather than nil
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		log.Printf("models: encode models: %v", err)
	}
}
