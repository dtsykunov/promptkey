package main

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

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type streamEvent struct {
	chunk string
	err   error
}

func streamCompletion(ctx context.Context, p Provider, userMsg string) <-chan streamEvent {
	ch := make(chan streamEvent, 16)
	go func() {
		defer close(ch)

		var messages []chatMessage
		if p.SystemPrompt != "" {
			messages = append(messages, chatMessage{Role: "system", Content: p.SystemPrompt})
		}
		messages = append(messages, chatMessage{Role: "user", Content: userMsg})

		body, err := json.Marshal(chatRequest{
			Model:    p.Model,
			Messages: messages,
			Stream:   true,
		})
		if err != nil {
			ch <- streamEvent{err: fmt.Errorf("marshal request: %w", err)}
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			ch <- streamEvent{err: fmt.Errorf("create request: %w", err)}
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if p.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ch <- streamEvent{err: fmt.Errorf("http: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errBody, _ := io.ReadAll(resp.Body)
			ch <- streamEvent{err: fmt.Errorf("api error %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[len("data: "):]
			if data == "[DONE]" {
				return
			}
			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- streamEvent{chunk: chunk.Choices[0].Delta.Content}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- streamEvent{err: fmt.Errorf("scanner: %w", err)}
		}
	}()
	return ch
}
