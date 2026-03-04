// mockserver is a standalone mock OpenAI-compatible streaming server for
// testing PromptKey without a real AI provider.
//
// Usage:
//
//	go run ./cmd/mockserver [-addr :11435]
//
// Then set baseURL to "http://localhost:11435/v1" in config.json.
// Use model name "error" to trigger an HTTP 500 response.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

var lorem = strings.Fields(
	"Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod " +
		"tempor incididunt ut labore et dolore magna aliqua Ut enim ad minim " +
		"veniam quis nostrud exercitation ullamco laboris nisi ut aliquip ex " +
		"ea commodo consequat Duis aute irure dolor in reprehenderit in " +
		"voluptate velit esse cillum dolore eu fugiat nulla pariatur",
)

type incomingRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type chunkResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req incomingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var firstUser string
	for _, m := range req.Messages {
		if m.Role == "user" {
			firstUser = m.Content
			break
		}
	}
	log.Printf("model=%q message=%q", req.Model, firstUser)

	if req.Model == "error" {
		http.Error(w, "mock error: model is \"error\"", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for _, word := range lorem {
		var chunk chunkResponse
		chunk.Choices = []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		}{{}}
		chunk.Choices[0].Delta.Content = word + " "
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Millisecond):
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

type modelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID     string `json:"id"`
		Object string `json:"object"`
	} `json:"data"`
}

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := modelsResponse{Object: "list"}
	for _, name := range []string{"mock-gpt-4o", "mock-gpt-3.5-turbo", "mock-claude", "error"} {
		resp.Data = append(resp.Data, struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		}{ID: name, Object: "model"})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	addr := flag.String("addr", ":11435", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handler)
	mux.HandleFunc("/v1/models", modelsHandler)

	log.Printf("mock server listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
