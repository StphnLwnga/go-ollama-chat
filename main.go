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

	// Build the routing table: each model name maps to the provider that serves it.
	router := ai.NewRouter()
	router.Register(ai.Ollama{}, ai.DefaultModel)
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		router.Register(ai.Groq{APIKey: key},
			"openai/gpt-oss-20b",
			"openai/gpt-oss-120b",
			"qwen/qwen3.8-27b",
		)
	} else {
		log.Println("GROQ_API_KEY not set: Groq models are off")
	}
	ai.Use(router)
	log.Printf("AI models: %v", router.Models())

	mux := http.NewServeMux()

	// Static files like CSS, JS, images served from ./static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Application routes — add yours here
	mux.HandleFunc("GET /{$}", handlers.Index)
	mux.HandleFunc("POST /chat", handlers.Chat)      // streaming chat endpoint
	mux.HandleFunc("GET /history", handlers.History) // saved conversation (E6)
	mux.HandleFunc("GET /models", handlers.Models)   // model names the router serves

	log.Println("🚀 Server running → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
