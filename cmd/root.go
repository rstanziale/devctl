package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/config"
	"github.com/rs-box/devctl/internal/logger"
	"github.com/rs-box/devctl/internal/plugin"
)

var flagConfigPath string
var flagConcurrencyMode string
var flagMaxConcurrency int
var flagDryRun bool
var flagVerboseCount int
var cfgLoaded *config.Config
var cfgLoadedPath string
var cfgInitErr error

var rootCmd = &cobra.Command{
	Use:           "devctl",
	Short:         "Developer Environment Manager",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return cfgInitErr
	},
}

var _ config.Provider = getConfig
var _ config.FlagsProvider = getGlobalFlags

func init() {
	cobra.OnInitialize(initializeApplication)

	rootCmd.CompletionOptions.DisableDefaultCmd = false

	rootCmd.PersistentFlags().StringVar(&flagConfigPath, "config", "./configs/config.yaml", "config file (default is ./configs/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&flagConcurrencyMode, "concurrency", "", "Concurrency mode: sequential, parallel, semaphore (default: config value or parallel)")
	rootCmd.PersistentFlags().IntVar(&flagMaxConcurrency, "max-concurrency", 0, "Max concurrent checks (used with semaphore mode)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be done without making changes")
	rootCmd.PersistentFlags().CountVarP(&flagVerboseCount, "verbose", "v", "Increase log verbosity (-v for debug, -vv for trace)")
}

func Execute(ctx context.Context) {
	registerPlugins()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.GetLogger(logger.WithComponent("Command")).Error("command failed", "error", err)

		var exitErr *plugin.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		os.Exit(1)
	}
}

// registerPlugins discovers external plugin binaries in the "plugins" directory
// (relative to the current working directory) and registers each one as a
// subcommand of the root command. Built-in commands are never shadowed.
//
// This runs before cobra parses CLI flags (registered subcommands must exist
// prior to argument dispatch), so the plugins directory cannot be sourced
// from a --flag; it is fixed relative to the working directory.
func registerPlugins() {
	log := logger.GetLogger(logger.WithComponent("Plugins"))
	pluginsDir := "plugins"

	plugins, err := plugin.Discover(pluginsDir)
	if err != nil {
		log.Warn("failed to discover plugins", "dir", pluginsDir, "error", err)
		return
	}

	existing := make(map[string]struct{})
	for _, sub := range rootCmd.Commands() {
		existing[sub.Name()] = struct{}{}
	}

	for _, p := range plugins {
		if _, conflict := existing[p.Name]; conflict {
			log.Warn("plugin conflicts with a built-in command and will be skipped", "plugin", p.Name)
			continue
		}
		rootCmd.AddCommand(p.ToCobraCommand())
	}
}

// Initializes the application by loading the configuration and setting up logging
func initializeApplication() {
	logger.Init(flagVerboseCount)

	cfg, usedPath, err := config.Load(flagConfigPath)
	if err != nil {
		cfgInitErr = err
		return
	}

	cfgLoaded = cfg
	cfgLoadedPath = usedPath
	cfgInitErr = nil
}

// Helper function to get the loaded configuration and its file path
func getConfig() (*config.Config, string) {
	return cfgLoaded, cfgLoadedPath
}

// Helper function to get the current values of devctl's persistent CLI flags
func getGlobalFlags() config.GlobalFlags {
	return config.GlobalFlags{
		ConcurrencyMode: flagConcurrencyMode,
		MaxConcurrency:  flagMaxConcurrency,
		DryRun:          flagDryRun,
		VerboseCount:    flagVerboseCount,
	}
}
