package ai

import (
	"strings"
	"testing"

	"github.com/rs-box/devctl/internal/ai/ollama"
	"github.com/rs-box/devctl/internal/config"
)

func ollamaCfg() config.AISettings {
	return config.AISettings{
		Enabled: true,
		Engine:  config.AIEngineOllama,
		Ollama: config.OllamaSettings{
			Model:   "gemma4:e4b",
			BaseURL: "http://localhost:11434",
		},
	}
}

func TestNewClientOllamaReturnsOllamaClient(t *testing.T) {
	client, err := NewClient(ollamaCfg())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*ollama.Client); !ok {
		t.Fatalf("expected *ollama.Client, got %T", client)
	}
}

func TestNewClientUnknownEngineReturnsError(t *testing.T) {
	cfg := ollamaCfg()
	cfg.Engine = config.AIEngine("unsupported")

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for unknown engine, got nil")
	}
	if !strings.Contains(err.Error(), "unknown AI engine") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewClientEmptyEngineReturnsError(t *testing.T) {
	cfg := ollamaCfg()
	cfg.Engine = config.AIEngine("")

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for empty engine, got nil")
	}
	if !strings.Contains(err.Error(), "unknown AI engine") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
