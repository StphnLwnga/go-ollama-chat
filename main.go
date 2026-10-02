package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/StphnLwnga/go-ollama-chat/ai"
	"github.com/StphnLwnga/go-ollama-chat/config"
	"github.com/StphnLwnga/go-ollama-chat/db"
	"github.com/StphnLwnga/go-ollama-chat/handlers"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if it exists. Variables already set in the environment win over the file.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf("reading .env: %v", err)
	}

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		log.Fatalf("invalid configuration:\n%v", err)
	}
	log.Printf("config: %+v", cfg) // the Groq key prints as [redacted], or [not set] when empty

	// Open (or create) the SQLite file the conversation is persisted to.
	if err := db.Open(cfg.DBPath); err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	// Build the routing table: each model name maps to the provider that serves it.
	router := ai.NewRouter()
	// First-byte budgets come from measurements: a local cold start from disk plus a long
	// prefill took about 18s; a hosted provider answers in about a second.
	router.Register(ai.Ollama{BaseURL: cfg.OllamaURL, HTTPClient: ai.NewHTTPClient(time.Minute)}, cfg.OllamaModel)
	if cfg.GroqAPIKey != "" {
		router.Register(ai.Groq{APIKey: string(cfg.GroqAPIKey), HTTPClient: ai.NewHTTPClient(15 * time.Second)},
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

	// Install HTTP routes defined in routes()
	for _, r := range routes() {
		mux.Handle(r.pattern, r.handler)
	}

	log.Printf("🚀 Server running → http://localhost%s", cfg.Addr())
	log.Fatal(newServer(cfg.Addr(), mux).ListenAndServe())
}

// newServer returns the HTTP server, with timeouts against slow and idle clients.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,  // a client that sends headers slowly cannot hold a connection (Slowloris)
		ReadTimeout:       30 * time.Second, // the whole request, body included
		IdleTimeout:       2 * time.Minute,  // a kept-alive connection with no new request
		// No WriteTimeout: it covers the whole response, so it would cut off every
		// streamed answer. The chat handler sets a deadline for each event instead.
	}
}

// route is one HTTP route: a ServeMux pattern and its handler.
type route struct {
	pattern string
	handler http.Handler
}

// routes lists every HTTP route the server registers. The docs test reads it too.
func routes() []route {
	return []route{
		{"GET /{$}", http.HandlerFunc(handlers.Index)},
		{"POST /chat", http.HandlerFunc(handlers.Chat)},
		{"GET /history", http.HandlerFunc(handlers.History)},
		{"GET /models", http.HandlerFunc(handlers.Models)},
		{"/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static")))},
	}
}
