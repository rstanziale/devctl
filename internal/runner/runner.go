package runner

import (
	"context"
	"sync"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
)

// Command represents a task that can be run by the Runner
type Command interface {
	Name() string
	Run(ctx context.Context) error
}

// Result represents the outcome of running a Command
type Result struct {
	Name  string
	Error error
}

// Runner executes a list of Commands with configurable concurrency settings
type Runner struct {
	commands        []Command
	concurrencyMode config.ConcurrencyMode
	maxConcurrency  int
}

var log = logger.GetLogger(logger.WithComponent("Runner"))

// defaultMaxConcurrency is used when no explicit concurrency limit is provided.
const defaultMaxConcurrency = 5

// NewRunner creates a Runner with default settings (parallel mode, max concurrency 5)
func NewRunner(commands []Command) *Runner {
	return &Runner{
		commands:        commands,
		concurrencyMode: config.ModeParallel,
		maxConcurrency:  defaultMaxConcurrency,
	}
}

// NewRunnerWithConfig creates a Runner with specified concurrency mode and max concurrency
func NewRunnerWithConfig(commands []Command, mode config.ConcurrencyMode, maxConcurrency int) *Runner {
	r := &Runner{
		commands:        commands,
		concurrencyMode: mode,
		maxConcurrency:  maxConcurrency,
	}

	if maxConcurrency <= 0 {
		r.maxConcurrency = defaultMaxConcurrency
	}

	return r
}

// GetConcurrencyMode determines the concurrency mode: CLI flag has priority over config, defaults to parallel
func GetConcurrencyMode(flagCli string, flagConfig config.ConcurrencyMode) config.ConcurrencyMode {
	if flagCli != "" {
		return config.ParseConcurrencyMode(flagCli)
	}
	return config.ParseConcurrencyMode(string(flagConfig))
}

// GetConcurrencySlots determines max concurrency: CLI flag has priority over config, defaults to 5
func GetConcurrencySlots(slotCli int, slotConfig int) int {
	if slotCli != 0 {
		return slotCli
	}
	if slotConfig != 0 {
		return slotConfig
	}
	return defaultMaxConcurrency
}

// ReportResults logs the outcome of each result: an error for failures, info for successes.
func ReportResults(results []Result) {
	for _, r := range results {
		if r.Error != nil {
			log.Error("command failed", "name", r.Name, "error", r.Error)
		} else {
			log.Info("command succeeded", "name", r.Name)
		}
	}
}

// AllSucceeded reports whether every result completed without error.
func AllSucceeded(results []Result) bool {
	for _, r := range results {
		if r.Error != nil {
			return false
		}
	}
	return true
}

// RunAll executes all commands according to the configured concurrency mode and returns their results
func (r *Runner) RunAll(ctx context.Context) []Result {
	switch r.concurrencyMode {
	case config.ModeSequential:
		return r.runAllSequential(ctx)
	case config.ModeParallel:
		return r.runAllParallel(ctx)
	case config.ModeSemaphore:
		return r.runAllSemaphore(ctx)
	default:
		return r.runAllParallel(ctx)
	}
}

// runAllSequential runs commands one after another, waiting for each to finish before starting the next
func (r *Runner) runAllSequential(ctx context.Context) []Result {
	var results []Result

	for _, c := range r.commands {
		// Short-circuit on cancellation: record the remaining commands as
		// cancelled without executing them, so a Ctrl+C stops further work
		// instead of running every command against a dead context.
		if err := ctx.Err(); err != nil {
			results = append(results, Result{
				Name:  c.Name(),
				Error: err,
			})
			continue
		}

		err := c.Run(ctx)

		results = append(results, Result{
			Name:  c.Name(),
			Error: err,
		})
	}

	return results
}

// runAllParallel runs all commands concurrently and collects their results
func (r *Runner) runAllParallel(ctx context.Context) []Result {
	var wg sync.WaitGroup
	resultsChan := make(chan Result, len(r.commands))

	for _, c := range r.commands {
		wg.Add(1)
		go func(command Command) {
			defer wg.Done()
			err := command.Run(ctx)
			resultsChan <- Result{
				Name:  command.Name(),
				Error: err,
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []Result
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// runAllSemaphore runs commands with a limit on the number of concurrent executions using a semaphore pattern
func (r *Runner) runAllSemaphore(ctx context.Context) []Result {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, r.maxConcurrency)
	resultsChan := make(chan Result, len(r.commands))

	for _, c := range r.commands {
		wg.Add(1)
		go func(command Command) {
			defer wg.Done()

			// Bail out immediately if the context is already cancelled, so a
			// cancelled run does not start any new command. This also makes
			// cancellation deterministic: without this check, select could
			// still pick a ready semaphore slot over ctx.Done().
			if err := ctx.Err(); err != nil {
				resultsChan <- Result{
					Name:  command.Name(),
					Error: err,
				}
				return
			}

			// Acquire a slot, but stay responsive to cancellation: if the
			// context is cancelled while waiting for a free slot, abandon the
			// command instead of blocking until a slot opens.
			select {
			case semaphore <- struct{}{}: // Take a slot
				defer func() { <-semaphore }() // Release the slot
				if err := ctx.Err(); err != nil {
					resultsChan <- Result{
						Name:  command.Name(),
						Error: err,
					}
					return
				}
			case <-ctx.Done():
				resultsChan <- Result{
					Name:  command.Name(),
					Error: ctx.Err(),
				}
				return
			}

			err := command.Run(ctx)
			resultsChan <- Result{
				Name:  command.Name(),
				Error: err,
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []Result
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}
