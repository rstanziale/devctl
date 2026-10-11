package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs-box/devctl/internal/config"
)

func TestGenerateStreamsResponseChunksAndSuppressesThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathGenerate {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var request generateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !request.Stream {
			t.Fatal("expected stream to be enabled")
		}
		if request.Think {
			t.Fatal("expected thinking to be disabled")
		}

		_, _ = io.WriteString(w, `{"thinking":"internal reasoning"}`+"\n")
		_, _ = io.WriteString(w, `{"response":"Hello"}`+"\n")
		_, _ = io.WriteString(w, `{"response":" world"}`+"\n")
		_, _ = io.WriteString(w, `{"done":true}`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	var chunks []string
	err := client.Generate(context.Background(), "test prompt", nil, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}
	if got, want := strings.Join(chunks, ""), "Hello world"; got != want {
		t.Fatalf("response = %q, want %q", got, want)
	}
}

func TestGenerateInvokesOnStartBeforeFirstChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"response":"Hello"}`+"\n")
		_, _ = io.WriteString(w, `{"done":true}`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})

	var events []string
	onStart := func() { events = append(events, "start") }
	onChunk := func(chunk string) error {
		events = append(events, "chunk:"+chunk)
		return nil
	}

	if err := client.Generate(context.Background(), "test prompt", onStart, onChunk); err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}

	if len(events) == 0 || events[0] != "start" {
		t.Fatalf("expected onStart to fire before any chunk, got events: %v", events)
	}

	startCount := 0
	for _, e := range events {
		if e == "start" {
			startCount++
		}
	}
	if startCount != 1 {
		t.Fatalf("expected onStart to fire exactly once, got %d: %v", startCount, events)
	}
}

func TestGenerateToleratesNilOnStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"response":"Hello"}`+"\n")
		_, _ = io.WriteString(w, `{"done":true}`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return nil })
	if err != nil {
		t.Fatalf("Generate returned an error: %v", err)
	}
}

func TestGenerateRequiresTerminalDoneEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"response":"partial"}`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return nil })
	if err == nil {
		t.Fatal("expected an error for incomplete stream")
	}
	if !strings.Contains(err.Error(), "before completion") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateReturnsCallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"response":"answer"}`+"\n")
		_, _ = io.WriteString(w, `{"done":true}`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	want := errors.New("writer failed")
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapped %v", err, want)
	}
}

func TestGenerateReturnsOllamaStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, `{"error":"model unavailable","done":true}`)
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateReturnsMalformedStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"response":`+"\n")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "decode stream response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "Ollama is starting")
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	err := client.Generate(context.Background(), "test prompt", nil, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateHonorsContextCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		time.Sleep(time.Second)
	}))
	defer server.Close()

	client := NewClient(config.OllamaSettings{Model: "test-model", BaseURL: server.URL})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errorsCh := make(chan error, 1)
	go func() {
		errorsCh <- client.Generate(ctx, "test prompt", nil, func(string) error { return nil })
	}()
	<-requestStarted
	cancel()

	err := <-errorsCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want wrapped context cancellation", err)
	}
}

func TestGenerateExplainsOverallTimeoutAfterPartialResponse(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = fmt.Fprintln(w, `{"response":"partial answer"}`)
		w.(http.Flusher).Flush()
		<-release
	}))
	defer server.Close()
	defer close(release)

	client := NewClient(config.OllamaSettings{
		Model: "test-model", BaseURL: server.URL, Timeout: time.Second,
	})
	var output strings.Builder
	err := client.Generate(context.Background(), "test prompt", nil, func(chunk string) error {
		_, err := output.WriteString(chunk)
		return err
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want wrapped deadline exceeded", err)
	}
	if output.String() != "partial answer" {
		t.Fatalf("expected partial streamed response, got %q", output.String())
	}
	for _, want := range []string{"overall request timeout (1s)", "settings.ai.ollama.timeout"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

