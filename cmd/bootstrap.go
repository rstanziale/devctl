package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/bootstrap"
	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/runner"
	"github.com/rs-box/devctl/internal/timing"
)

func init() {
	rootCmd.AddCommand(newBootstrapCommand(getConfig))
}

// newBootstrapCommand builds the "bootstrap" command. cfgProvider is used to
// access the loaded configuration at run time.
func newBootstrapCommand(cfgProvider config.Provider) *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the development environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBootstrap(cmd, cfgProvider)
		},
	}
}

// Main logic for the bootstrap command
func runBootstrap(cmd *cobra.Command, cfgProvider config.Provider) error {
	log := logger.GetLogger(logger.WithComponent("BootstrapCommand"))

	done := timing.LogCommandTiming("bootstrap")
	defer done()

	cfg, filename := cfgProvider()

	log.Info("config loaded", "file", filename)

	mode := runner.GetConcurrencyMode(flagConcurrencyMode, cfg.ConcurrencyMode)
	maxConcurrency := runner.GetConcurrencySlots(flagMaxConcurrency, cfg.MaxConcurrency)

	log.Info(
		"starting bootstrap",
		"repos", len(cfg.Repos),
		"workspace", cfg.Workspace,
		"mode", mode,
		"maxConcurrency", maxConcurrency,
	)

	if flagDryRun {
		log.Info("dry run mode", "repos", len(cfg.Repos))

		for _, repo := range cfg.Repos {
			dest := filepath.Join(cfg.Workspace, repo.Dir)
			log.Info("planned clone", "repo", repo.Name, "url", repo.URL, "destination", dest, "gitflow", repo.Gitflow)
		}

		return nil
	}

	// Create commands for each repo
	var commands []runner.Command
	for _, repo := range cfg.Repos {
		commands = append(commands, bootstrap.NewRepoCloneCommand(repo, cfg.Workspace))
	}

	// Create and run the runner
	r := runner.NewRunnerWithConfig(commands, mode, maxConcurrency)

	results := r.RunAll(cmd.Context())
	runner.ReportResults(results)

	if !runner.AllSucceeded(results) {
		return fmt.Errorf("bootstrap completed with errors")
	}

	return nil
}
