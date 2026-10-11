package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/doctor"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/runner"
	"github.com/rs-box/devctl/internal/timing"
)

func init() {
	rootCmd.AddCommand(newDoctorCommand(getConfig))
}

// newDoctorCommand builds the "doctor" command. cfgProvider is used to access
// the loaded configuration at run time.
func newDoctorCommand(cfgProvider config.Provider) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate configuration and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, cfgProvider)
		},
	}
}

// Main logic for the doctor command
func runDoctor(cmd *cobra.Command, cfgProvider config.Provider) error {
	log := logger.GetLogger(logger.WithComponent("DoctorCommand"))

	done := timing.LogCommandTiming("doctor")
	defer done()

	cfg, filename := cfgProvider()

	log.Info("config loaded", "file", filename)

	if err := config.Validate(cfg); err != nil {
		log.Error("config validation failed", "error", err)
		return fmt.Errorf("config validation failed: %w", err)
	}

	concurrencyMode := runner.GetConcurrencyMode(flagConcurrencyMode, cfg.ConcurrencyMode)
	maxConcurrency := runner.GetConcurrencySlots(flagMaxConcurrency, cfg.MaxConcurrency)

	log.Info(
		"starting doctor",
		"healthchecks", len(cfg.Healthchecks),
		"tools", len(cfg.Tools),
		"mode", concurrencyMode,
		"maxConcurrency", maxConcurrency,
	)

	if flagDryRun {
		log.Info("dry run mode", "healthchecks", len(cfg.Healthchecks), "tools", len(cfg.Tools))

		for _, hc := range cfg.Healthchecks {
			log.Info("planned check", "type", "url", "name", hc.Name, "target", hc.URL, "protocol", hc.Type, "timeout", hc.Timeout)
		}

		for _, tool := range cfg.Tools {
			log.Info("planned check", "type", "tool", "name", tool.Name, "command", tool.Command)
		}

		return nil
	}

	// Create checks based on config
	var checks []runner.Command

	// Add URL checks from config
	client := &http.Client{}
	for _, hc := range cfg.Healthchecks {
		checks = append(checks, doctor.NewURLCheckCommand(hc, client))
	}

	// Add tool checks from config
	for _, tool := range cfg.Tools {
		checks = append(checks, doctor.NewToolCheckCommand(tool))
	}

	log.Info(
		"prepared checks",
		"total", len(checks),
		"url", len(cfg.Healthchecks),
		"tools", len(cfg.Tools),
	)

	// Create and run the runner
	r := runner.NewRunnerWithConfig(checks, concurrencyMode, maxConcurrency)

	results := r.RunAll(cmd.Context())
	runner.ReportResults(results)

	if !runner.AllSucceeded(results) {
		return fmt.Errorf("doctor completed with errors")
	}

	return nil
}
