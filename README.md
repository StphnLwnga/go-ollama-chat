# go-ollama-chat

A real-time, streaming AI chat app: **Go** on the backend, a **local LLM via [Ollama](https://ollama.com)** for inference, and a dependency-light frontend. Replies stream into the browser token-by-token over Server-Sent Events. Runs fully local and private by default. Groq is optional.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Ollama](https://img.shields.io/badge/LLM-Ollama%20(local)-000000)
![SQLite](https://img.shields.io/badge/storage-SQLite-003B57?logo=sqlite&logoColor=white)

## Features

- **Live token streaming**: replies appear as they're generated, via SSE (`text/event-stream`) with per-token HTTP flushing.
- **Polished chat UI**: auto-growing prompt box, role-based message bubbles, auto-scroll, and a scroll-to-bottom control.
- **Markdown rendering**: assistant replies render as formatted markdown (code blocks, lists) once complete, sanitized against XSS (via `marked` + `DOMPurify`).
- **Stop generation**: cancel a streaming reply mid-flight (browser `AbortController` → the server stops cleanly).
- **Local and private by default**: with Ollama, prompts never leave your machine and no API key is needed. Groq is an optional hosted provider (see "Choosing the AI provider").
- **Persistent**: conversations are saved to SQLite and restored on refresh.
- **Lean backend**: Go standard library plus one pure-Go SQLite driver (no CGO, no C toolchain).

## Stack

| Layer     | Tech                            |
|-----------|---------------------------------|
| Backend   | Go (`net/http`, stdlib)         |
| LLM       | Ollama (`llama3.2:3b` default)  |
| Transport | Server-Sent Events (SSE)        |
| Storage   | SQLite (pure-Go `modernc.org/sqlite`) |
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

## Choosing the AI provider

The app uses a local Ollama model by default. To use [Groq](https://groq.com) instead, set two environment variables:

| Variable       | Value                                                   |
|----------------|---------------------------------------------------------|
| `AI_PROVIDER`  | `ollama` (default) or `groq`                            |
| `GROQ_API_KEY` | your Groq API key; required when `AI_PROVIDER=groq`     |

With `AI_PROVIDER=groq` and no key, the app refuses to start and says why.

The app does not read `.env` yet, so load it into the shell first:

```bash
set -a; . ./.env; set +a; AI_PROVIDER=groq go run .
```

The models Groq offers depend on the account. To list the models your key can use:

```bash
curl -s https://api.groq.com/openai/v1/models -H "Authorization: Bearer $GROQ_API_KEY"
```

**Known limitation:** the provider is chosen once per process, but the model is chosen per message. The model in the dropdown must match the running provider: a local model name sent to Groq returns a 404. The planned fix is to route each message to a provider by its model name.

## Measured results

Same prompt, three runs each, measured on 2026-09-29. Local runs used an Apple M4 with 16 GB of memory; Groq runs on Groq's own hardware.

| Run                              | First token    | Total          |
|----------------------------------|----------------|----------------|
| local llama3.2:3b, cold start    | 5.6 s          | 7.8 s          |
| local llama3.2:3b, warm          | 0.03 to 0.05 s | 2.3 to 2.5 s   |
| Groq gpt-oss-20b                 | 0.39 to 0.53 s | 0.49 to 0.71 s |
| Groq, network only (error reply) | 0.17 to 0.25 s | same           |

The local model returns the first token sooner, but Groq finishes about 15 times faster. The answers have different lengths, so tokens per second is the fair comparison; the app does not count tokens yet.

To repeat a run, start the app and replace `<model>` with a model name:

```bash
curl -s -N -o /dev/null -X POST localhost:8080/chat -H 'Content-Type: application/json' -d '{"model":"<model>","messages":[{"role":"user","content":"Explain what an HTTP status code is in three sentences."}]}' -w 'first token %{time_starttransfer}s, total %{time_total}s\n'
```

## Project layout

```bash
main.go                HTTP server + route registration
ai/provider.go         Provider interface + startup selection; handlers call only this package
ai/ollama.go           Ollama provider (local)
ai/groq.go             Groq provider (hosted, needs GROQ_API_KEY)
db/db.go               SQLite persistence: saves and loads the conversation
handlers/chat.go       The SSE streaming chat endpoint
handlers/history.go    Serves the saved conversation as JSON
handlers/handlers.go   Page handler
templates/index.html   Chat UI
static/                style.css + chat.js
```
