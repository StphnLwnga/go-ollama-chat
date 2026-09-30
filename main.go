package main

import (
	"log"
	"net/http"
	"os"

	"github.com/StphnLwnga/go-ollama-chat/ai"
	"github.com/StphnLwnga/go-ollama-chat/config"
	"github.com/StphnLwnga/go-ollama-chat/db"
	"github.com/StphnLwnga/go-ollama-chat/handlers"
)

func main() {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		log.Fatalf("invalid configuration:\n%v", err)
	}
	log.Printf("config: %+v", cfg) // the Groq key prints as [redacted]

	// Open (or create) the SQLite file the conversation is persisted to.
	if err := db.Open(cfg.DBPath); err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	// Build the routing table: each model name maps to the provider that serves it.
	router := ai.NewRouter()
	router.Register(ai.Ollama{BaseURL: cfg.OllamaURL}, cfg.OllamaModel)
	if cfg.GroqAPIKey != "" {
		router.Register(ai.Groq{APIKey: string(cfg.GroqAPIKey)},
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

	log.Printf("🚀 Server running → http://localhost%s", cfg.Addr())
	log.Fatal(http.ListenAndServe(cfg.Addr(), mux))
}
