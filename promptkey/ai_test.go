package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sseServer(words []string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != http.StatusOK {
			http.Error(w, "internal server error", statusCode)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, word := range words {
			chunk := streamChunk{}
			chunk.Choices = []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			}{{}}
			chunk.Choices[0].Delta.Content = word
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
}

func TestStreamCompletion_success(t *testing.T) {
	words := []string{"Hello", " ", "world"}
	srv := sseServer(words, http.StatusOK)
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Model: "test"}
	ch := streamCompletion(context.Background(), p, "hi")

	var got []string
	for ev := range ch {
		if ev.err != nil {
			t.Fatalf("unexpected error: %v", ev.err)
		}
		got = append(got, ev.chunk)
	}
	if strings.Join(got, "") != "Hello world" {
		t.Errorf("got %q, want %q", strings.Join(got, ""), "Hello world")
	}
}

func TestStreamCompletion_httpError(t *testing.T) {
	srv := sseServer(nil, http.StatusInternalServerError)
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Model: "test"}
	ch := streamCompletion(context.Background(), p, "hi")

	var gotErr error
	for ev := range ch {
		if ev.err != nil {
			gotErr = ev.err
		}
	}
	if gotErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(gotErr.Error(), "500") {
		t.Errorf("expected 500 in error, got: %v", gotErr)
	}
}

func TestStreamCompletion_cancel(t *testing.T) {
	// Server that streams slowly
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 100; i++ {
			chunk := streamChunk{}
			chunk.Choices = []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			}{{}}
			chunk.Choices[0].Delta.Content = "word "
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	p := Provider{BaseURL: srv.URL, Model: "test"}
	ch := streamCompletion(ctx, p, "hi")

	// Cancel after receiving at least one chunk
	<-ch
	cancel()

	// Drain — must finish cleanly (no hang)
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("channel did not close after context cancel")
	}
}

func TestStreamCompletion_requestFormat(t *testing.T) {
	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Model: "my-model"}
	for range streamCompletion(context.Background(), p, "test message") {
	}

	if captured.Model != "my-model" {
		t.Errorf("model = %q, want %q", captured.Model, "my-model")
	}
	if !captured.Stream {
		t.Error("stream should be true")
	}
	if len(captured.Messages) == 0 {
		t.Fatal("messages should not be empty")
	}
	last := captured.Messages[len(captured.Messages)-1]
	if last.Role != "user" || last.Content != "test message" {
		t.Errorf("last message = {%q %q}, want {user test message}", last.Role, last.Content)
	}
}

func TestStreamCompletion_systemPrompt(t *testing.T) {
	var captured chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	t.Run("with system prompt", func(t *testing.T) {
		p := Provider{BaseURL: srv.URL, Model: "m", SystemPrompt: "Be concise."}
		for range streamCompletion(context.Background(), p, "hi") {
		}
		if len(captured.Messages) < 2 {
			t.Fatalf("expected 2 messages, got %d", len(captured.Messages))
		}
		if captured.Messages[0].Role != "system" {
			t.Errorf("first message role = %q, want system", captured.Messages[0].Role)
		}
		if captured.Messages[0].Content != "Be concise." {
			t.Errorf("system content = %q, want %q", captured.Messages[0].Content, "Be concise.")
		}
	})

	t.Run("without system prompt", func(t *testing.T) {
		p := Provider{BaseURL: srv.URL, Model: "m", SystemPrompt: ""}
		for range streamCompletion(context.Background(), p, "hi") {
		}
		if len(captured.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(captured.Messages))
		}
		if captured.Messages[0].Role != "user" {
			t.Errorf("first message role = %q, want user", captured.Messages[0].Role)
		}
	})
}
