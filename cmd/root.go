package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"stackpilot/internal/utils"
)

// dockpilotURL is shared by all stack subcommands.
var dockpilotURL string

var rootCmd = &cobra.Command{
	Use:   "stackpilot",
	Short: "Stack orchestrator — deploys multi-service stacks via dockpilot",
	Long: `stackpilot reads YAML stack files, validates structure, resolves dependency
order, and calls the dockpilot REST API to deploy or remove services.

stackpilot never talks to Docker directly — it delegates all container
execution to dockpilot.

Example:
  dockpilot server --port 8088
  stackpilot stack deploy examples/backend.yaml --dockpilot-url http://127.0.0.1:8088`,

	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.PrintError(err.Error())
		os.Exit(1)
	}
}
