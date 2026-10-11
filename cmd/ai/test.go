// Package aicmd collects the second-level subcommands of the "ai" command.
package aicmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/ai"
	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/timing"
)

// NewTestCommand builds the "ai test" subcommand. cfgProvider is used to
// access the loaded configuration at run time.
func NewTestCommand(cfgProvider config.Provider) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test the AI engine connection",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAITest(cmd, cfgProvider)
		},
	}
}

func runAITest(cmd *cobra.Command, cfgProvider config.Provider) error {
	log := logger.GetLogger(logger.WithComponent("AITestCommand"))

	done := timing.LogCommandTiming("ai-test")
	defer done()

	cfg, filename := cfgProvider()

	log.Info("config loaded", "file", filename)

	if !cfg.Settings.AI.Enabled {
		return fmt.Errorf("AI is not enabled in configuration")
	}

	client, err := ai.NewClient(cfg.Settings.AI)
	if err != nil {
		return fmt.Errorf("create AI client: %w", err)
	}

	if _, err := streamGeneration(cmd, client, "Say hi in one short sentence."); err != nil {
		return fmt.Errorf("AI request failed: %w", err)
	}

	return nil
}
