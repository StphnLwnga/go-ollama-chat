package main

import (
	"log"
	"net/http"
	"os"

	"github.com/StphnLwnga/go-ollama-chat/ai"
	"github.com/StphnLwnga/go-ollama-chat/db"
	"github.com/StphnLwnga/go-ollama-chat/handlers"
)

func main() {
	// Open (or create) the SQLite file the conversation is persisted to.
	if err := db.Open("chat.db"); err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}
	if provider == "groq" && os.Getenv("GROQ_API_KEY") == "" {
		log.Fatalf("AI_PROVIDER=groq needs GROQ_API_KEY to be set")
	}
	ai.Init(provider, os.Getenv("GROQ_API_KEY"))
	log.Printf("🚀 Using AI provider: %s", provider)

	mux := http.NewServeMux()

	// Static files like CSS, JS, images served from ./static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Application routes — add yours here
	mux.HandleFunc("GET /{$}", handlers.Index)
	mux.HandleFunc("POST /chat", handlers.Chat)      // streaming chat endpoint
	mux.HandleFunc("GET /history", handlers.History) // saved conversation (E6)

	log.Println("🚀 Server running → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
