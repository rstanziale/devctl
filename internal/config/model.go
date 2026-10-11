package config

import "time"

// Provider returns the loaded configuration and the path it was read from.
type Provider func() (*Config, string)

// GlobalFlags holds the values of devctl's persistent CLI flags.
type GlobalFlags struct {
	ConcurrencyMode string
	MaxConcurrency  int
	DryRun          bool
	VerboseCount    int
}

// FlagsProvider returns the current values of devctl's persistent CLI flags.
type FlagsProvider func() GlobalFlags

type ConcurrencyMode string
type TaskMode string
type AIEngine string
type HealthcheckType string

const (
	ModeSequential ConcurrencyMode = "sequential"
	ModeParallel   ConcurrencyMode = "parallel"
	ModeSemaphore  ConcurrencyMode = "semaphore"
)

const (
	TaskModeSequential TaskMode = "sequential"
	TaskModeParallel   TaskMode = "parallel"
)

const (
	AIEngineOllama AIEngine = "ollama"
)

const (
	HealthcheckHTTP HealthcheckType = "http"
	HealthcheckTCP  HealthcheckType = "tcp"
)

type Config struct {
	Version         int             `mapstructure:"version"`
	Workspace       string          `mapstructure:"workspace"`
	ConcurrencyMode ConcurrencyMode `mapstructure:"concurrency_mode"`
	MaxConcurrency  int             `mapstructure:"max_concurrency"`
	Healthchecks    []Healthcheck   `mapstructure:"healthchecks"`
	Tools           []Tool          `mapstructure:"tools"`
	Repos           []Repo          `mapstructure:"repos"`
	Tasks           []Task          `mapstructure:"tasks"`
	Settings        Settings        `mapstructure:"settings"`
}

type Settings struct {
	AI AISettings `mapstructure:"ai"`
}

type AISettings struct {
	Enabled bool           `mapstructure:"enabled"`
	Engine  AIEngine       `mapstructure:"engine"`
	Ollama  OllamaSettings `mapstructure:"ollama"`
}

type OllamaSettings struct {
	Model   string        `mapstructure:"model"`
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type Healthcheck struct {
	Name    string          `mapstructure:"name"`
	URL     string          `mapstructure:"url"`
	Type    HealthcheckType `mapstructure:"type"`
	Timeout time.Duration   `mapstructure:"timeout"`
}

type Tool struct {
	Name    string `mapstructure:"name"`
	Command string `mapstructure:"command"`
}

type Repo struct {
	Name    string `mapstructure:"name"`
	URL     string `mapstructure:"url"`
	Dir     string `mapstructure:"dir"`
	Gitflow bool   `mapstructure:"gitflow"`
}

type Task struct {
	Name     string    `mapstructure:"name"`
	Mode     TaskMode  `mapstructure:"mode"`
	Subtasks []Subtask `mapstructure:"subtasks"`
}

type Subtask struct {
	Name     string   `mapstructure:"name"`
	Dir      string   `mapstructure:"dir"`
	Commands []string `mapstructure:"commands"`
}
