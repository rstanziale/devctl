package cmd

import (
	"github.com/spf13/cobra"

	aicmd "github.com/rs-box/devctl/cmd/ai"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered commands",
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(aicmd.NewTestCommand(getConfig))
	aiCmd.AddCommand(aicmd.NewLogAnalyzerCommand(getConfig, getGlobalFlags))
}
