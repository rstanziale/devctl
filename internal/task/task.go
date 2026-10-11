package task

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/runner"
)

var log = logger.GetLogger(logger.WithComponent("Task"))

type SubtaskCommand struct {
	taskName string
	subtask  config.Subtask
	workDir  string
}

func NewSubtaskCommand(taskName string, subtask config.Subtask, workspace string) *SubtaskCommand {
	return &SubtaskCommand{
		taskName: taskName,
		subtask:  subtask,
		workDir:  resolveWorkingDir(workspace, subtask.Dir),
	}
}

func (c *SubtaskCommand) Name() string {
	return fmt.Sprintf("%s/%s", c.taskName, c.subtask.Name)
}

func (c *SubtaskCommand) Run(ctx context.Context) error {
	for _, command := range c.subtask.Commands {
		// Stop before each command if the context has been cancelled, so a
		// cancellation between commands does not start further work.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("subtask %s cancelled: %w", c.subtask.Name, err)
		}

		log.Debug("running task command", "task", c.taskName, "subtask", c.subtask.Name, "dir", c.workDir, "command", command)

		if err := runShellCommand(ctx, c.workDir, command); err != nil {
			return fmt.Errorf("subtask %s failed: %w", c.subtask.Name, err)
		}
	}

	return nil
}

func FindTask(cfg *config.Config, taskName string) (*config.Task, error) {
	for i := range cfg.Tasks {
		if cfg.Tasks[i].Name == taskName {
			return &cfg.Tasks[i], nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskName)
}

func BuildRunnerCommands(task config.Task, workspace string) []runner.Command {
	commands := make([]runner.Command, 0, len(task.Subtasks))

	for _, subtask := range task.Subtasks {
		commands = append(commands, NewSubtaskCommand(task.Name, subtask, workspace))
	}

	return commands
}

func RunnerMode(task config.Task) config.ConcurrencyMode {
	if config.ParseTaskMode(string(task.Mode)) == config.TaskModeParallel {
		return config.ModeParallel
	}

	return config.ModeSequential
}

func resolveWorkingDir(workspace string, subtaskDir string) string {
	if filepath.IsAbs(subtaskDir) {
		return subtaskDir
	}

	if workspace == "" {
		return subtaskDir
	}

	return filepath.Join(workspace, subtaskDir)
}

// runShellCommand executes the given command in a shell, using the specified working directory.
// It captures both stdout and stderr, and returns an error if the command fails, including the output for debugging.
// The command is executed using "cmd /C" on Windows and "sh -c" on Unix-like systems to ensure proper shell behavior.
func runShellCommand(ctx context.Context, dir string, command string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the failure is due to context cancellation/timeout, surface that
		// cause instead of the opaque process exit error. exec kills the
		// process on cancellation, which otherwise only yields an exit status.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("command %q cancelled: %w", command, ctxErr)
		}

		trimmedOutput := strings.TrimSpace(string(output))
		if trimmedOutput == "" {
			return fmt.Errorf("command %q failed: %w", command, err)
		}

		return fmt.Errorf("command %q failed: %w (output: %s)", command, err, trimmedOutput)
	}

	return nil
}
