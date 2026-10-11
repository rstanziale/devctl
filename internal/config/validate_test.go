package config

import (
	"strings"
	"testing"
	"time"
)

var validURL string = "https://example.com"
var invalidURL string = "://invalid-url"

func validConfig() *Config {
	return &Config{
		Version:   1,
		Workspace: "test_workspace",
		Healthchecks: []Healthcheck{
			{
				Name:    "api",
				URL:     validURL,
				Type:    "http",
				Timeout: time.Second,
			},
		},
		Tools: []Tool{
			{Name: "tool1", Command: "cmd1"},
		},
		Repos: []Repo{
			{Name: "repo1", URL: validURL, Dir: "/dir1"},
		},
	}
}

func validAISettings() AISettings {
	return AISettings{
		Enabled: true,
		Engine:  AIEngineOllama,
		Ollama: OllamaSettings{
			Model:   "gemma4:e4b",
			BaseURL: "http://localhost:11434",
		},
	}
}

func TestConfigCheckValidConfig(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Healthchecks: []Healthcheck{
			{
				Name:    "api",
				URL:     validURL,
				Type:    "http",
				Timeout: time.Second,
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error for a valid config: %v", err)
	}
}

func TestConfigCheckInvalidURL(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Healthchecks: []Healthcheck{
			{
				Name:    "api",
				URL:     invalidURL,
				Type:    "http",
				Timeout: time.Second,
			},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for invalid URL format, got nil")
	}
}

func TestValidateDuplicateURLs(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Healthchecks: []Healthcheck{
			{
				Name:    "api",
				URL:     validURL,
				Type:    "http",
				Timeout: time.Second,
			},
			{
				Name:    "api",
				URL:     validURL,
				Type:    "http",
				Timeout: time.Second,
			},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for duplicate healthcheck names, got nil")
	}
}

func TestConfigCheckEmptyToolsAndRepos(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Tools:     []Tool{}, // empty
		Repos:     []Repo{}, // empty
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error for empty tools and repos slices: %v", err)
	}
}

func TestConfigCheckDuplicateTools(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Tools: []Tool{
			{Name: "tool1", Command: "cmd1"},
			{Name: "tool1", Command: "cmd2"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for duplicate tool names, got nil")
	}
}

func TestConfigCheckDuplicateRepos(t *testing.T) {
	cfg := &Config{
		Version:   1,
		Workspace: "test_workspace",
		Repos: []Repo{
			{Name: "repo1", URL: validURL, Dir: "/dir1"},
			{Name: "repo1", URL: validURL, Dir: "/dir2"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for duplicate repo names, got nil")
	}
}

func TestConfigCheckConcurrencySemaphoreRequiresMaxConcurrency(t *testing.T) {
	cfg := &Config{
		Version:         1,
		Workspace:       "test_workspace",
		ConcurrencyMode: ModeSemaphore,
		MaxConcurrency:  0,
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error: semaphore mode requires max_concurrency > 0, got nil")
	}
}

func TestConfigCheckConcurrencyNonSemaphoreDoesNotRequireMaxConcurrency(t *testing.T) {
	cases := []ConcurrencyMode{"", ModeSequential, ModeParallel}

	for _, mode := range cases {
		t.Run(string(mode), func(t *testing.T) {
			cfg := &Config{
				Version:         1,
				Workspace:       "test_workspace",
				ConcurrencyMode: mode,
				MaxConcurrency:  0,
			}

			if err := Validate(cfg); err != nil {
				t.Fatalf("unexpected error for mode %q: %v", mode, err)
			}
		})
	}
}

func TestConfigCheckConcurrencyModeInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.ConcurrencyMode = ConcurrencyMode("invalid")

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid concurrency_mode value, got nil")
	}
	if !strings.Contains(err.Error(), "concurrency_mode is invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckVersionRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Version = 0

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: version must be 1, got nil")
	}
	if !strings.Contains(err.Error(), "version must be 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckVersionInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Version = 2

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: version must be 1, got nil")
	}
	if !strings.Contains(err.Error(), "version must be 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckWorkspaceRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Workspace = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: workspace is required, got nil")
	}
	if !strings.Contains(err.Error(), "workspace is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckNameRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Name = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: healthcheck name is required, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckURLRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].URL = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: healthcheck url is required, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: url is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckTypeRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Type = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: healthcheck type is required, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: type is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckTypeInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Type = "udp"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: healthcheck type is not a valid value, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: type must be one of: http, tcp") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckHTTPURLInvalidForType(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Type = "http"
	cfg.Healthchecks[0].URL = "localhost:8080"

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: http type requires a valid http URL, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: invalid http url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckTCPEndpointInvalidForType(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Type = "tcp"
	cfg.Healthchecks[0].URL = validURL

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: tcp type requires a valid host:port endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: invalid tcp endpoint") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckTCPEndpointValid(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Type = "tcp"
	cfg.Healthchecks[0].URL = "localhost:8080"

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckHealthcheckTimeoutMustBePositive(t *testing.T) {
	cfg := validConfig()
	cfg.Healthchecks[0].Timeout = 0

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: healthcheck timeout must be positive, got nil")
	}
	if !strings.Contains(err.Error(), "healthchecks[0]: timeout must be > 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckToolNameRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Tools[0].Name = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: tool name is required, got nil")
	}
	if !strings.Contains(err.Error(), "tools[0]: name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckToolCommandRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Tools[0].Command = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: tool command is required, got nil")
	}
	if !strings.Contains(err.Error(), "tools[0]: command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckRepoNameRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Repos[0].Name = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: repo name is required, got nil")
	}
	if !strings.Contains(err.Error(), "repos[0]: name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckRepoURLRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Repos[0].URL = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: repo url is required, got nil")
	}
	if !strings.Contains(err.Error(), "repos[0]: url is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckRepoInvalidURL(t *testing.T) {
	cfg := validConfig()
	cfg.Repos[0].URL = invalidURL

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: repo URL is not a valid URL, got nil")
	}
	if !strings.Contains(err.Error(), "repos[0]: invalid url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckRepoDirRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Repos[0].Dir = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error: repo dir is required, got nil")
	}
	if !strings.Contains(err.Error(), "repos[0]: dir is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckTaskValid(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = []Task{
		{
			Name: "build-fe",
			Mode: TaskModeParallel,
			Subtasks: []Subtask{
				{Name: "app-1", Dir: "/dir/app-1", Commands: []string{"npm install", "npm run build"}},
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error for valid task: %v", err)
	}
}

func TestConfigCheckTaskModeInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = []Task{
		{
			Name: "build-fe",
			Mode: "invalid",
			Subtasks: []Subtask{
				{Name: "app-1", Dir: "/dir/app-1", Commands: []string{"npm install"}},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid task mode, got nil")
	}
	if !strings.Contains(err.Error(), "mode must be one of: parallel, sequential") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckTaskSubtaskCommandsRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks = []Task{
		{
			Name: "build-fe",
			Mode: TaskModeSequential,
			Subtasks: []Subtask{
				{Name: "app-1", Dir: "/dir/app-1", Commands: []string{}},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing commands, got nil")
	}
	if !strings.Contains(err.Error(), "at least one command is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckAIDisabledSkipsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = AISettings{Enabled: false}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error when AI is disabled: %v", err)
	}
}

func TestConfigCheckAIValidOllama(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = validAISettings()

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error for valid AI config: %v", err)
	}
}

func TestConfigCheckAIEngineRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = validAISettings()
	cfg.Settings.AI.Engine = AIEngine("")

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error when engine is empty, got nil")
	}
	if !strings.Contains(err.Error(), "settings.ai.engine is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckAIEngineInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = validAISettings()
	cfg.Settings.AI.Engine = AIEngine("openai")

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for unknown engine, got nil")
	}
	if !strings.Contains(err.Error(), "settings.ai.engine is invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckAIOllamaBaseURLRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = validAISettings()
	cfg.Settings.AI.Ollama.BaseURL = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error when ollama base_url is empty, got nil")
	}
	if !strings.Contains(err.Error(), "settings.ai.ollama.base_url is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigCheckAIOllamaModelRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Settings.AI = validAISettings()
	cfg.Settings.AI.Ollama.Model = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error when ollama model is empty, got nil")
	}
	if !strings.Contains(err.Error(), "settings.ai.ollama.model is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
