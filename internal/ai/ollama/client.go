package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
)

const (
	pathGenerate   = "/api/generate"
	defaultTimeout = 5 * time.Minute
	maxBodyPrint   = 150
)

// Client implements ai.Client for a locally running Ollama instance.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Ollama client from the provided settings.
func NewClient(cfg config.OllamaSettings) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Think  bool   `json:"think"`
}

type generateResponse struct {
	Response string `json:"response"`
	Thinking string `json:"thinking"`
	Done     bool   `json:"done"`
	Error    string `json:"error"`
}

// Generate sends a prompt to Ollama and delivers visible response chunks as
// they arrive. onStart is invoked exactly once, before the request is sent
// and before the first chunk is decoded, so it fires strictly after all
// request-preparation logging above has already happened.
func (c *Client) Generate(ctx context.Context, prompt string, onStart func(), onChunk func(string) error) error {
	log := logger.GetLogger(logger.WithComponent("GenerateOllama"))

	body, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: true,
		Think:  false,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + pathGenerate
	log.Debug("Url and request body", "url", url, "body", truncate(string(body), maxBodyPrint))

	if onStart != nil {
		onStart()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.requestError(ctx, "send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, errBody)
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var result generateResponse
		if err := decoder.Decode(&result); err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("ollama stream ended before completion")
			}
			return c.requestError(ctx, "decode stream response", err)
		}

		if result.Error != "" {
			return fmt.Errorf("ollama stream error: %s", result.Error)
		}

		if result.Response != "" {
			if err := onChunk(result.Response); err != nil {
				return fmt.Errorf("handle response chunk: %w", err)
			}
		}

		if result.Done {
			return nil
		}
	}
}

func (c *Client) requestError(ctx context.Context, operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("ollama request canceled: %w", err)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", operation, err)
	}

	if ctx.Err() != nil {
		return fmt.Errorf("ollama request context deadline exceeded: %w", err)
	}

	if c.httpClient.Timeout > 0 {
		return fmt.Errorf("ollama generation exceeded the overall request timeout (%s); increase settings.ai.ollama.timeout: %w", c.httpClient.Timeout, err)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

// truncate shortens s to at most limit characters, appending "..." when truncated.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
