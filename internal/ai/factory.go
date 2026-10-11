package ai

import (
	"context"
	"fmt"

	"github.com/rs-box/devctl/internal/ai/ollama"
	"github.com/rs-box/devctl/internal/config"
)

// Client is the common interface for all AI providers.
// Implementations are created via NewClient.
type Client interface {
	// Generate sends a prompt to the underlying LLM. onStart is invoked at
	// most once, after the request has been accepted and before the first
	// chunk is decoded, so callers can defer UI feedback (e.g. a spinner)
	// until streaming actually begins instead of racing pre-flight logging.
	// onChunk is called for each response chunk.
	Generate(ctx context.Context, prompt string, onStart func(), onChunk func(string) error) error
}

// NewClient creates an AI Client for the engine specified in cfg.
// The only supported engine at the moment is "ollama".
func NewClient(cfg config.AISettings) (Client, error) {
	switch cfg.Engine {
	case config.AIEngineOllama:
		return ollama.NewClient(cfg.Ollama), nil
	default:
		return nil, fmt.Errorf("unknown AI engine: %q", cfg.Engine)
	}
}
