package runner

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
)

type fakeCommand struct {
	name  string
	runFn func(ctx context.Context) error
}

func (f fakeCommand) Name() string {
	return f.name
}

func (f fakeCommand) Run(ctx context.Context) error {
	if f.runFn == nil {
		return nil
	}
	return f.runFn(ctx)
}

func TestNewRunnerDefaults(t *testing.T) {
	r := NewRunner(nil)

	if r.concurrencyMode != config.ModeParallel {
		t.Fatalf("expected mode %q, got %q", config.ModeParallel, r.concurrencyMode)
	}

	if r.maxConcurrency != 5 {
		t.Fatalf("expected maxConcurrency 5, got %d", r.maxConcurrency)
	}
}

func TestNewRunnerWithConfigInvalidMaxConcurrency(t *testing.T) {
	r := NewRunnerWithConfig(nil, config.ModeSemaphore, 0)

	if r.maxConcurrency != 5 {
		t.Fatalf("expected fallback maxConcurrency 5, got %d", r.maxConcurrency)
	}

	if r.concurrencyMode != config.ModeSemaphore {
		t.Fatalf("expected mode %q, got %q", config.ModeSemaphore, r.concurrencyMode)
	}
}

func TestGetConcurrencyModePriorityAndDefault(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		cfg      config.ConcurrencyMode
		expected config.ConcurrencyMode
	}{
		{
			name:     "cli has priority",
			cli:      "sequential",
			cfg:      config.ModeParallel,
			expected: config.ModeSequential,
		},
		{
			name:     "use config when cli is empty",
			cli:      "",
			cfg:      config.ModeSemaphore,
			expected: config.ModeSemaphore,
		},
		{
			name:     "invalid mode defaults to parallel",
			cli:      "",
			cfg:      "unknown",
			expected: config.ModeParallel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mode := GetConcurrencyMode(tc.cli, tc.cfg)
			if mode != tc.expected {
				t.Fatalf("expected mode %q, got %q", tc.expected, mode)
			}
		})
	}
}

func TestGetConcurrencySlotsPriorityAndDefault(t *testing.T) {
	tests := []struct {
		name     string
		cli      int
		cfg      int
		expected int
	}{
		{
			name:     "cli has priority",
			cli:      2,
			cfg:      7,
			expected: 2,
		},
		{
			name:     "use config when cli is zero",
			cli:      0,
			cfg:      3,
			expected: 3,
		},
		{
			name:     "default to five when both are zero",
			cli:      0,
			cfg:      0,
			expected: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slots := GetConcurrencySlots(tc.cli, tc.cfg)
			if slots != tc.expected {
				t.Fatalf("expected slots %d, got %d", tc.expected, slots)
			}
		})
	}
}

func TestRunAllSequentialPreservesOrderAndErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	commands := []Command{
		fakeCommand{name: "first"},
		fakeCommand{name: "second", runFn: func(context.Context) error { return expectedErr }},
		fakeCommand{name: "third"},
	}

	r := NewRunnerWithConfig(commands, config.ModeSequential, 1)
	results := r.RunAll(context.Background())

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Name != "first" || results[1].Name != "second" || results[2].Name != "third" {
		t.Fatalf("sequential execution should preserve order, got %v", []string{results[0].Name, results[1].Name, results[2].Name})
	}

	if results[1].Error == nil || results[1].Error.Error() != expectedErr.Error() {
		t.Fatalf("expected second result error %v, got %v", expectedErr, results[1].Error)
	}
}

func TestRunAllParallelRunsAllCommands(t *testing.T) {
	commands := []Command{
		fakeCommand{name: "a"},
		fakeCommand{name: "b"},
		fakeCommand{name: "c", runFn: func(context.Context) error { return errors.New("c failed") }},
	}

	r := NewRunnerWithConfig(commands, config.ModeParallel, 5)
	results := r.RunAll(context.Background())

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	byName := map[string]error{}
	for _, res := range results {
		byName[res.Name] = res.Error
	}

	if _, ok := byName["a"]; !ok {
		t.Fatalf("missing result for command a")
	}
	if _, ok := byName["b"]; !ok {
		t.Fatalf("missing result for command b")
	}
	if err, ok := byName["c"]; !ok {
		t.Fatalf("missing result for command c")
	} else if err == nil {
		t.Fatalf("expected error for command c")
	}
}

func TestRunAllSemaphoreRespectsMaxConcurrency(t *testing.T) {
	var inFlight int32
	var maxInFlight int32

	updateMax := func(current int32) {
		for {
			prev := atomic.LoadInt32(&maxInFlight)
			if current <= prev {
				return
			}
			if atomic.CompareAndSwapInt32(&maxInFlight, prev, current) {
				return
			}
		}
	}

	commands := make([]Command, 0, 12)
	for i := 0; i < 12; i++ {
		commands = append(commands, fakeCommand{
			name: "cmd",
			runFn: func(context.Context) error {
				current := atomic.AddInt32(&inFlight, 1)
				updateMax(current)
				time.Sleep(25 * time.Millisecond)
				atomic.AddInt32(&inFlight, -1)
				return nil
			},
		})
	}

	r := NewRunnerWithConfig(commands, config.ModeSemaphore, 3)
	results := r.RunAll(context.Background())

	if len(results) != 12 {
		t.Fatalf("expected 12 results, got %d", len(results))
	}

	if maxInFlight > 3 {
		t.Fatalf("expected max concurrent executions <= 3, got %d", maxInFlight)
	}
}

func TestRunAllInvalidModeFallsBackToParallel(t *testing.T) {
	commands := []Command{
		fakeCommand{name: "x"},
		fakeCommand{name: "y"},
	}

	r := NewRunnerWithConfig(commands, config.ConcurrencyMode("invalid"), 2)
	results := r.RunAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestRunAllPropagatesContextToCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	commands := []Command{
		fakeCommand{
			name: "ctx-aware",
			runFn: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
	}

	r := NewRunnerWithConfig(commands, config.ModeSequential, 1)
	results := r.RunAll(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !errors.Is(results[0].Error, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", results[0].Error)
	}
}

func TestRunAllSequentialShortCircuitsOnCanceledContext(t *testing.T) {
	var executed int32

	ctx, cancel := context.WithCancel(context.Background())

	commands := []Command{
		fakeCommand{
			name: "first",
			runFn: func(context.Context) error {
				atomic.AddInt32(&executed, 1)
				// Cancel the context from within the first command so that
				// the runner must stop before invoking the remaining ones.
				cancel()
				return nil
			},
		},
		fakeCommand{name: "second", runFn: func(context.Context) error { atomic.AddInt32(&executed, 1); return nil }},
		fakeCommand{name: "third", runFn: func(context.Context) error { atomic.AddInt32(&executed, 1); return nil }},
	}

	r := NewRunnerWithConfig(commands, config.ModeSequential, 1)
	results := r.RunAll(ctx)

	if got := atomic.LoadInt32(&executed); got != 1 {
		t.Fatalf("expected only the first command to execute, got %d executions", got)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (one per command), got %d", len(results))
	}

	// The first command succeeded before cancellation.
	if results[0].Name != "first" || results[0].Error != nil {
		t.Fatalf("expected first command to succeed, got name=%q err=%v", results[0].Name, results[0].Error)
	}

	// The remaining commands must be reported as canceled without running.
	for _, idx := range []int{1, 2} {
		if !errors.Is(results[idx].Error, context.Canceled) {
			t.Fatalf("expected result %d to be context.Canceled, got %v", idx, results[idx].Error)
		}
	}
}

func TestRunAllSemaphoreWakesBlockedGoroutinesOnCancel(t *testing.T) {
	// With a single slot and a pre-cancelled context, commands waiting to
	// acquire the semaphore must wake via ctx.Done() instead of blocking
	// forever. The whole call must return promptly.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var executed int32
	commands := make([]Command, 0, 8)
	for i := 0; i < 8; i++ {
		commands = append(commands, fakeCommand{
			name: "cmd",
			runFn: func(context.Context) error {
				atomic.AddInt32(&executed, 1)
				return nil
			},
		})
	}

	r := NewRunnerWithConfig(commands, config.ModeSemaphore, 1)

	done := make(chan []Result, 1)
	go func() { done <- r.RunAll(ctx) }()

	select {
	case results := <-done:
		if len(results) != 8 {
			t.Fatalf("expected 8 results, got %d", len(results))
		}
		for i, res := range results {
			if !errors.Is(res.Error, context.Canceled) {
				t.Fatalf("expected result %d to be context.Canceled, got %v", i, res.Error)
			}
		}
		if got := atomic.LoadInt32(&executed); got != 0 {
			t.Fatalf("expected no command to execute under cancelled context, got %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunAll did not return: semaphore goroutines blocked on cancellation")
	}
}

func TestReportResults(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	logger.SetOutput(&out, &errOut)
	t.Cleanup(func() {
		logger.SetOutput(nil, nil)
	})

	ReportResults([]Result{
		{Name: "ok", Error: nil},
		{Name: "ko", Error: errors.New("boom")},
	})

	stdout := out.String()
	if !strings.Contains(stdout, "command succeeded") || !strings.Contains(stdout, "name=ok") {
		t.Fatalf("expected success log on stdout, got: %s", stdout)
	}

	stderr := errOut.String()
	if !strings.Contains(stderr, "command failed") || !strings.Contains(stderr, "name=ko") || !strings.Contains(stderr, "boom") {
		t.Fatalf("expected failure log on stderr, got: %s", stderr)
	}
}

func TestAllSucceeded(t *testing.T) {
	allOK := AllSucceeded([]Result{{Name: "ok", Error: nil}})
	if !allOK {
		t.Fatalf("expected true when all results are successful")
	}

	notAllOK := AllSucceeded([]Result{{Name: "ok", Error: nil}, {Name: "ko", Error: errors.New("fail")}})
	if notAllOK {
		t.Fatalf("expected false when at least one result fails")
	}
}

func TestRunAllWithNoCommandsReturnsEmptyResults(t *testing.T) {
	r := NewRunnerWithConfig([]Command{}, config.ModeParallel, 2)
	results := r.RunAll(context.Background())

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestRunAllParallelIncludesEachCommandExactlyOnce(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{}

	commands := []Command{
		fakeCommand{name: "a", runFn: func(context.Context) error { mu.Lock(); hits["a"]++; mu.Unlock(); return nil }},
		fakeCommand{name: "b", runFn: func(context.Context) error { mu.Lock(); hits["b"]++; mu.Unlock(); return nil }},
		fakeCommand{name: "c", runFn: func(context.Context) error { mu.Lock(); hits["c"]++; mu.Unlock(); return nil }},
	}

	r := NewRunnerWithConfig(commands, config.ModeParallel, 3)
	results := r.RunAll(context.Background())

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, name := range []string{"a", "b", "c"} {
		if hits[name] != 1 {
			t.Fatalf("expected command %q to run once, got %d", name, hits[name])
		}
	}
}
