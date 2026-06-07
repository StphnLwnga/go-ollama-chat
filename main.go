package main

import (
	"log"
	"net/http"

	"github.com/StphnLwnga/go-ollama-chat/handlers"
)

func main() {
	mux := http.NewServeMux()

	// Static files like CSS, JS, images served from ./static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Application routes — add yours here
	mux.HandleFunc("/", handlers.Index)
	mux.HandleFunc("POST /chat", handlers.Chat) // Project 01: streaming chat endpoint

	log.Println("🚀 Server running → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
