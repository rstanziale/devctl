package task

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rs-box/devctl/internal/config"
)

func TestFindTaskFound(t *testing.T) {
	cfg := &config.Config{
		Tasks: []config.Task{
			{Name: "build-fe"},
			{Name: "build-be"},
		},
	}

	task, err := FindTask(cfg, "build-fe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Name != "build-fe" {
		t.Fatalf("expected task build-fe, got %s", task.Name)
	}
}

func TestFindTaskNotFound(t *testing.T) {
	cfg := &config.Config{Tasks: []config.Task{{Name: "other"}}}

	_, err := FindTask(cfg, "missing")
	if err == nil {
		t.Fatal("expected error for missing task")
	}

	if !strings.Contains(err.Error(), "task not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunnerMode(t *testing.T) {
	if mode := RunnerMode(config.Task{Mode: config.TaskModeSequential}); mode != config.ModeSequential {
		t.Fatalf("expected sequential mode, got %s", mode)
	}

	if mode := RunnerMode(config.Task{Mode: config.TaskModeParallel}); mode != config.ModeParallel {
		t.Fatalf("expected parallel mode, got %s", mode)
	}

	if mode := RunnerMode(config.Task{Mode: "invalid"}); mode != config.ModeSequential {
		t.Fatalf("expected fallback sequential mode, got %s", mode)
	}
}

func TestBuildRunnerCommands(t *testing.T) {
	taskCfg := config.Task{
		Name: "build-fe",
		Subtasks: []config.Subtask{
			{Name: "app-1", Dir: ".", Commands: []string{"echo one"}},
			{Name: "app-2", Dir: ".", Commands: []string{"echo two"}},
		},
	}

	commands := BuildRunnerCommands(taskCfg, "")
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	if commands[0].Name() != "build-fe/app-1" {
		t.Fatalf("unexpected first command name: %s", commands[0].Name())
	}
}

func TestSubtaskRunFailsOnInvalidCommand(t *testing.T) {
	cmd := NewSubtaskCommand("build-fe", config.Subtask{
		Name:     "broken",
		Dir:      ".",
		Commands: []string{"command_that_does_not_exist_42"},
	}, "")

	err := cmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected command execution error")
	}
}

func TestSubtaskRunShortCircuitsOnCanceledContext(t *testing.T) {
	// A pre-cancelled context must make Run report context.Canceled without
	// running the commands. This pins the cancellation contract of the loop.
	cmd := NewSubtaskCommand("build-fe", config.Subtask{
		Name:     "many-steps",
		Dir:      ".",
		Commands: []string{"echo one", "echo two", "echo three"},
	}, "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cmd.Run(ctx)
	if err == nil {
		t.Fatal("expected error when running with a cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSubtaskRunStopsWhenContextCancelledMidExecution(t *testing.T) {
	// A long-running first command is cancelled while it executes. Run must
	// report context.Canceled (the cancellation cause), not the opaque process
	// exit error, and must not proceed to later commands.
	var sleep string
	if runtime.GOOS == "windows" {
		// ~2s without external deps and without quoted-path redirection.
		sleep = "ping -n 3 127.0.0.1 >nul"
	} else {
		sleep = "sleep 2"
	}

	cmd := NewSubtaskCommand("build-fe", config.Subtask{
		Name:     "slow-then-more",
		Dir:      ".",
		Commands: []string{sleep, "echo should-not-run"},
	}, "")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := cmd.Run(ctx)
	if err == nil {
		t.Fatal("expected error when context is cancelled mid-execution")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestResolveWorkingDirAbsolute(t *testing.T) {
	abs := "/tmp/app"
	if runtime.GOOS == "windows" {
		abs = `C:\\temp\\app`
	}

	resolved := resolveWorkingDir("/workspace", abs)
	if resolved != abs {
		t.Fatalf("expected absolute dir %s, got %s", abs, resolved)
	}
}

func TestResolveWorkingDirRelativeWithWorkspace(t *testing.T) {
	workspace := filepath.Join("root", "workspace")
	relative := filepath.Join("apps", "app-1")

	resolved := resolveWorkingDir(workspace, relative)
	expected := filepath.Join(workspace, relative)
	if resolved != expected {
		t.Fatalf("expected %s, got %s", expected, resolved)
	}
}

func TestResolveWorkingDirRelativeWithoutWorkspace(t *testing.T) {
	relative := filepath.Join("apps", "app-1")

	resolved := resolveWorkingDir("", relative)
	if resolved != relative {
		t.Fatalf("expected %s, got %s", relative, resolved)
	}
}
