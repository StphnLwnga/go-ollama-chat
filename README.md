# go-ollama-chat

A real-time, streaming AI chat app — **Go** on the backend, a **local LLM via [Ollama](https://ollama.com)** for inference, and a polished, dependency-light frontend. Replies stream into the browser token-by-token over Server-Sent Events. No cloud, no API keys, fully private.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Ollama](https://img.shields.io/badge/LLM-Ollama%20(local)-000000)
![Backend](https://img.shields.io/badge/backend-stdlib%20only-success)

## Features

- **Live token streaming** — replies appear as they're generated, via SSE (`text/event-stream`) with per-token HTTP flushing.
- **Polished chat UI** — auto-growing prompt box, role-based message bubbles, auto-scroll, and a scroll-to-bottom control.
- **Markdown rendering** — assistant replies render as formatted markdown (code blocks, lists) once complete, sanitized against XSS (via `marked` + `DOMPurify`).
- **Stop generation** — cancel a streaming reply mid-flight (browser `AbortController` → the server stops cleanly).
- **100% local & private** — prompts never leave your machine. No API keys.
- **Tiny backend** — Go standard library only, zero third-party Go packages.

## Stack

| Layer     | Tech                            |
|-----------|---------------------------------|
| Backend   | Go 1.22 (`net/http`, stdlib)    |
| LLM       | Ollama (`llama3.2:3b` default)  |
| Transport | Server-Sent Events (SSE)        |
| Frontend  | Vanilla JS + CSS                |

## How it works

```bash
Browser  ──POST /chat──▶  Go server  ──stream:true──▶  Ollama
         ◀──SSE tokens──              ◀──NDJSON chunks──
```

The browser POSTs a message; the Go handler opens an SSE stream, requests a streaming completion from Ollama, and relays each token to the browser the instant it arrives. All LLM communication is isolated in the `ai/` package (`ai.Chat` / `ai.ChatStream`), so the provider can be swapped without touching any HTTP handler.

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Ollama](https://ollama.com), running locally

## Quick start

```bash
git clone https://github.com/StphnLwnga/go-ollama-chat
cd go-ollama-chat

# Pull the default model (first run only, ~2 GB)
ollama pull llama3.2:3b

# Run
make run            # or: go run main.go
# → http://localhost:8080
```

## Project layout

```bash
main.go                HTTP server + route registration
ai/ollama.go           Ollama client (Chat + ChatStream) — the only file that talks to the LLM
handlers/chat.go       The SSE streaming chat endpoint
handlers/handlers.go   Page handler
templates/index.html   Chat UI
static/                style.css + chat.js
```
