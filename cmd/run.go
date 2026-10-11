package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/runner"
	"github.com/rs-box/devctl/internal/task"
	"github.com/rs-box/devctl/internal/timing"
)

func init() {
	rootCmd.AddCommand(newRunCommand(getConfig))
}

// newRunCommand builds the "run" command. cfgProvider is used to access the
// loaded configuration at run time.
func newRunCommand(cfgProvider config.Provider) *cobra.Command {
	var flagTaskName string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a configured task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTask(cmd, cfgProvider, flagTaskName)
		},
	}

	cmd.Flags().StringVar(&flagTaskName, "task", "", "Task name to execute")

	return cmd
}

func runTask(cmd *cobra.Command, cfgProvider config.Provider, taskName string) error {
	if taskName == "" {
		return fmt.Errorf("--task is required")
	}

	log := logger.GetLogger(logger.WithComponent("RunCommand"))

	done := timing.LogCommandTiming("run")
	defer done()

	cfg, filename := cfgProvider()
	log.Info("config loaded", "file", filename)

	taskCfg, err := task.FindTask(cfg, taskName)
	if err != nil {
		return err
	}

	mode := task.RunnerMode(*taskCfg)
	log.Info("starting task", "task", taskCfg.Name, "mode", mode, "subtasks", len(taskCfg.Subtasks))

	if flagDryRun {
		for _, subtask := range taskCfg.Subtasks {
			log.Info("planned subtask", "task", taskCfg.Name, "subtask", subtask.Name, "dir", subtask.Dir, "commands", subtask.Commands)
		}
		return nil
	}

	commands := task.BuildRunnerCommands(*taskCfg, cfg.Workspace)

	// Create and run the runner
	r := runner.NewRunnerWithConfig(commands, mode, 0)

	results := r.RunAll(cmd.Context())
	runner.ReportResults(results)

	if !runner.AllSucceeded(results) {
		return fmt.Errorf("task %s completed with errors", taskCfg.Name)
	}

	return nil
}
