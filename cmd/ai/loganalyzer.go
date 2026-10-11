// Package aicmd collects the second-level subcommands of the "ai" command.
package aicmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/ai"
	"github.com/rs-box/devctl/internal/ai/loganalyzer"
	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/timing"
)

// cliTimeLayout is the timestamp layout accepted by the --start and --end flags
// (e.g. "20250414T11:20:00").
const cliTimeLayout = loganalyzer.LayoutCompact

// NewLogAnalyzerCommand builds the "ai log-analyzer" subcommand. cfgProvider
// accesses the loaded configuration and flagsProvider accesses devctl's
// persistent CLI flags (e.g. --dry-run), both at run time.
func NewLogAnalyzerCommand(cfgProvider config.Provider, flagsProvider config.FlagsProvider) *cobra.Command {
	var (
		flagDir   string
		flagStart string
		flagEnd   string
	)

	cmd := &cobra.Command{
		Use:   "log-analyzer",
		Short: "Analyze logs from multiple sources of the same solution using AI",
		Long: "Scan a directory of logs (HAR, JSON) coming from different sources " +
			"(client, gateway, server, ...), extract the WARN/ERROR events within a " +
			"time window and ask the AI engine to correlate them and suggest fixes.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLogAnalyzer(cmd, cfgProvider, flagsProvider, flagDir, flagStart, flagEnd)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", "", "Directory containing the log files to analyze")
	cmd.Flags().StringVar(&flagStart, "start", "", "Start of the time window (format: 20060102T15:04:05)")
	cmd.Flags().StringVar(&flagEnd, "end", "", "End of the time window (format: 20060102T15:04:05)")

	_ = cmd.MarkFlagRequired("dir")
	_ = cmd.MarkFlagRequired("start")
	_ = cmd.MarkFlagRequired("end")

	return cmd
}

func runLogAnalyzer(cmd *cobra.Command, cfgProvider config.Provider, flagsProvider config.FlagsProvider, dir, startStr, endStr string) error {
	log := logger.GetLogger(logger.WithComponent("LogAnalyzerCommand"))

	done := timing.LogCommandTiming("ai-log-analyzer")
	defer done()

	cfg, filename := cfgProvider()
	log.Info("config loaded", "file", filename)

	if !cfg.Settings.AI.Enabled {
		return fmt.Errorf("AI is not enabled in configuration")
	}

	start, err := time.Parse(cliTimeLayout, startStr)
	if err != nil {
		return fmt.Errorf("parse --start %q: %w", startStr, err)
	}

	end, err := time.Parse(cliTimeLayout, endStr)
	if err != nil {
		return fmt.Errorf("parse --end %q: %w", endStr, err)
	}

	if end.Before(start) {
		return fmt.Errorf("--end must not be before --start")
	}

	entries, err := loganalyzer.Collect(dir)
	if err != nil {
		return fmt.Errorf("collect logs: %w", err)
	}
	log.Info("collected logs", "count", len(entries))

	window := loganalyzer.Filter(entries, start, end)
	log.Info("filtered logs", "count", len(window), "start", startStr, "end", endStr)

	if len(window) == 0 {
		log.Info("no WARN/ERROR logs found in the given window")
		return nil
	}

	prompt := loganalyzer.BuildPrompt(window)
	log.Trace("built prompt", "prompt", prompt)

	if flagsProvider().DryRun {
		log.Info("dry run mode", "prompt_chars", len(prompt))
		return nil
	}

	promptPath, err := loganalyzer.SavePrompt(prompt, dir)
	if err != nil {
		return fmt.Errorf("save prompt: %w", err)
	}
	log.Info("prompt saved to file", "path", promptPath)

	client, err := ai.NewClient(cfg.Settings.AI)
	if err != nil {
		return fmt.Errorf("create AI client: %w", err)
	}

	response, genErr := streamGeneration(cmd, client, prompt)
	if genErr != nil {
		log.Error("AI request failed", "error", genErr, "path", promptPath)
		return fmt.Errorf("AI request failed: %w", genErr)
	}

	path, err := loganalyzer.SaveResponse(response, dir)
	if err != nil {
		return fmt.Errorf("save AI analysis: %w", err)
	}

	log.Info("AI response saved to file", "path", path)
	return nil
}
