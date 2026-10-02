package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// Groq implements the Provider interface
type Groq struct {
	APIKey   string
	Endpoint string
}

type groqText struct {
	Content string `json:"content"`
}

type groqChoice struct {
	Delta   groqText `json:"delta"`   // streaming: the neest piece of text
	Message groqText `json:"message"` // non streaming
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
	// Usage   groqUsage  `json:"usage"`
	// Model   string     `json:"model"`
	// ID      string     `json:"id"`
	// Created int64      `json:"created"`
	// Object  string     `json:"object"`
}

// type groqUsage struct {
// 	PromptTokens int `json:"prompt_tokens"`
// 	CompletionTokens int `json:"completion_tokens"`
// 	TotalTokens int `json:"total_tokens"`
// }

func (g Groq) post(model string, messages []Message, stream bool) (*http.Response, error) {
	body, err := json.Marshal(chatRequest{Model: model, Messages: messages, Stream: stream})
	if err != nil {
		return nil, fmt.Errorf("ai: marshal request: %w", err)
	}

	url := g.Endpoint
	if url == "" {
		url = groqEndpoint
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: groq unreachable: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ai: groq returned %s: %s", resp.Status, msg)
	}

	return resp, nil
}

func (g Groq) Chat(ctx context.Context, model string, messages []Message) (string, error) {
	resp, err := g.post(model, messages, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ai: groq decode: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("ai: groq unexpected empty choices")
	}
	return out.Choices[0].Message.Content, nil
}

func (g Groq) ChatStream(ctx context.Context, model string, messages []Message, onChunk func(string) error) error {
	resp, err := g.post(model, messages, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// when streaming, groq sends one JSON object per line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk groqResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		token := chunk.Choices[0].Delta.Content
		if token != "" {
			if err := onChunk(token); err != nil {
				return nil // client disconnected: normal, not an error
			}
		}
	}

	return scanner.Err()
}
