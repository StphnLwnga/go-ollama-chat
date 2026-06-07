package main

import (
	"log"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/db"
	"github.com/StphnLwnga/go-ollama-chat/handlers"
)

func main() {
	// Open (or create) the SQLite file the conversation is persisted to.
	if err := db.Open("chat.db"); err != nil {
		log.Fatalf("could not open database: %v", err)
	}

	mux := http.NewServeMux()

	// Static files like CSS, JS, images served from ./static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Application routes — add yours here
	mux.HandleFunc("/", handlers.Index)
	mux.HandleFunc("POST /chat", handlers.Chat)      // streaming chat endpoint
	mux.HandleFunc("GET /history", handlers.History) // saved conversation (E6)

	log.Println("🚀 Server running → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
