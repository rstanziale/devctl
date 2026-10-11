package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// Validate checks cfg for structural and semantic errors: required fields,
// duplicate names, valid enum values, and cross-field constraints (e.g.
// max_concurrency required when concurrency_mode is semaphore).
func Validate(cfg *Config) error {
	if err := checkVersion(cfg); err != nil {
		return err
	}

	if err := checkWorkspace(cfg); err != nil {
		return err
	}

	if err := checkConcurrency(cfg); err != nil {
		return err
	}

	if err := checkHealthchecks(cfg); err != nil {
		return err
	}

	if err := checkTools(cfg); err != nil {
		return err
	}

	if err := checkRepos(cfg); err != nil {
		return err
	}

	if err := checkTasks(cfg); err != nil {
		return err
	}

	if err := checkAISettings(cfg); err != nil {
		return err
	}

	return nil
}

func checkVersion(cfg *Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	return nil
}

func checkWorkspace(cfg *Config) error {
	if cfg.Workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	return nil
}

func checkConcurrency(cfg *Config) error {
	if cfg.ConcurrencyMode != "" && !IsValidConcurrencyMode(cfg.ConcurrencyMode) {
		return fmt.Errorf("concurrency_mode is invalid: %s", cfg.ConcurrencyMode)
	}

	if cfg.ConcurrencyMode == ModeSemaphore && cfg.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be > 0 when concurrency_mode is semaphore")
	}
	return nil
}

// validateUniqueNamed validates a slice of named items, enforcing that each
// item has a non-empty, unique name before running kind-specific checks via
// rest. nameOf must return an error (typically "<kind>s[i]: name is required")
// when the item's name is missing; validateUniqueNamed does not hardcode that
// message so each caller keeps its own indexed error format.
func validateUniqueNamed[T any](
	items []T,
	kind string,
	nameOf func(i int, item T) (string, error),
	rest func(i int, item T) error,
) error {
	seen := make(map[string]struct{})

	for i, item := range items {
		name, err := nameOf(i, item)
		if err != nil {
			return err
		}

		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate %s name: %s", kind, name)
		}
		seen[name] = struct{}{}

		if err := rest(i, item); err != nil {
			return err
		}
	}

	return nil
}

func checkHealthchecks(cfg *Config) error {
	return validateUniqueNamed(cfg.Healthchecks, "healthcheck",
		func(i int, hc Healthcheck) (string, error) {
			if hc.Name == "" {
				return "", fmt.Errorf("healthchecks[%d]: name is required", i)
			}
			return hc.Name, nil
		},
		func(i int, hc Healthcheck) error {
			if hc.URL == "" {
				return fmt.Errorf("healthchecks[%d]: url is required", i)
			}

			if hc.Type == "" {
				return fmt.Errorf("healthchecks[%d]: type is required", i)
			}

			if err := validateHealthcheckURLByType(hc.Type, hc.URL); err != nil {
				return fmt.Errorf("healthchecks[%d]: %w", i, err)
			}

			if hc.Timeout <= 0 {
				return fmt.Errorf("healthchecks[%d]: timeout must be > 0", i)
			}

			return nil
		},
	)
}

func checkTools(cfg *Config) error {
	return validateUniqueNamed(cfg.Tools, "tool",
		func(i int, tool Tool) (string, error) {
			if tool.Name == "" {
				return "", fmt.Errorf("tools[%d]: name is required", i)
			}
			return tool.Name, nil
		},
		func(i int, tool Tool) error {
			if tool.Command == "" {
				return fmt.Errorf("tools[%d]: command is required", i)
			}
			return nil
		},
	)
}

func checkRepos(cfg *Config) error {
	return validateUniqueNamed(cfg.Repos, "repo",
		func(i int, repo Repo) (string, error) {
			if repo.Name == "" {
				return "", fmt.Errorf("repos[%d]: name is required", i)
			}
			return repo.Name, nil
		},
		func(i int, repo Repo) error {
			if repo.URL == "" {
				return fmt.Errorf("repos[%d]: url is required", i)
			}

			if _, err := url.ParseRequestURI(repo.URL); err != nil {
				return fmt.Errorf("repos[%d]: invalid url: %s", i, repo.URL)
			}

			if repo.Dir == "" {
				return fmt.Errorf("repos[%d]: dir is required", i)
			}

			return nil
		},
	)
}

func checkTasks(cfg *Config) error {
	return validateUniqueNamed(cfg.Tasks, "task",
		func(i int, task Task) (string, error) {
			if task.Name == "" {
				return "", fmt.Errorf("tasks[%d]: name is required", i)
			}
			return task.Name, nil
		},
		checkTask,
	)
}

func checkTask(i int, task Task) error {
	if task.Mode == "" {
		return fmt.Errorf("tasks[%d]: mode is required", i)
	}

	if !IsValidTaskMode(task.Mode) {
		return fmt.Errorf("tasks[%d]: mode must be one of: parallel, sequential", i)
	}

	if len(task.Subtasks) == 0 {
		return fmt.Errorf("tasks[%d]: at least one subtask is required", i)
	}

	return validateUniqueNamed(task.Subtasks, fmt.Sprintf("task %d subtask", i),
		func(j int, subtask Subtask) (string, error) {
			if subtask.Name == "" {
				return "", fmt.Errorf("tasks[%d].subtasks[%d]: name is required", i, j)
			}
			return subtask.Name, nil
		},
		func(j int, subtask Subtask) error {
			return checkSubtask(i, j, subtask)
		},
	)
}

func checkSubtask(i, j int, subtask Subtask) error {
	if subtask.Dir == "" {
		return fmt.Errorf("tasks[%d].subtasks[%d]: dir is required", i, j)
	}

	if len(subtask.Commands) == 0 {
		return fmt.Errorf("tasks[%d].subtasks[%d]: at least one command is required", i, j)
	}

	for k, command := range subtask.Commands {
		if command == "" {
			return fmt.Errorf("tasks[%d].subtasks[%d].commands[%d]: command is required", i, j, k)
		}
	}

	return nil
}

func checkAISettings(cfg *Config) error {
	ai := cfg.Settings.AI

	if !ai.Enabled {
		return nil
	}

	if ai.Engine == "" {
		return fmt.Errorf("settings.ai.engine is required when ai is enabled")
	}

	if !IsValidAIEngine(ai.Engine) {
		return fmt.Errorf("settings.ai.engine is invalid: %s", ai.Engine)
	}

	switch ai.Engine {
	case AIEngineOllama:
		if ai.Ollama.BaseURL == "" {
			return fmt.Errorf("settings.ai.ollama.base_url is required")
		}
		if ai.Ollama.Model == "" {
			return fmt.Errorf("settings.ai.ollama.model is required")
		}
	}

	return nil
}

// validateHealthcheckURLByType validates the healthcheck URL based on its type.
func validateHealthcheckURLByType(hcType HealthcheckType, rawURL string) error {
	switch hcType {
	case HealthcheckHTTP:
		u, err := url.ParseRequestURI(rawURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("invalid http url: %s", rawURL)
		}
		return nil
	case HealthcheckTCP:
		host, port, err := net.SplitHostPort(rawURL)
		if err != nil || host == "" {
			return fmt.Errorf("invalid tcp endpoint: %s", rawURL)
		}

		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("invalid tcp endpoint: %s", rawURL)
		}
		return nil
	default:
		return fmt.Errorf("type must be one of: http, tcp")
	}
}
