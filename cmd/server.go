package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"stackpilot/internal/server"
	"stackpilot/internal/utils"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the stackpilot HTTP API server",
	Long: `Start the stackpilot HTTP API server.

Exposes a REST API so tools like n8n can deploy stacks by POSTing YAML.
All container operations are delegated to dockpilot.

Endpoints:
  GET  /health            — liveness check
  POST /v1/stacks/deploy  — deploy a stack (JSON: {"yaml":"<stack YAML>"})
  POST /v1/stacks/remove  — remove a stack (JSON: {"yaml":"...","volumes":false})`,
	RunE: func(cmd *cobra.Command, args []string) error {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		url, _ := cmd.Flags().GetString("dockpilot-url")
		addr := fmt.Sprintf("%s:%d", host, port)
		utils.PrintInfo(fmt.Sprintf("stackpilot API server listening on http://%s", addr))
		utils.PrintInfo(fmt.Sprintf("dockpilot URL: %s", url))
		utils.PrintInfo("Press Ctrl+C to stop")
		fmt.Println()
		return server.Run(addr, url)
	},
}

func init() {
	serverCmd.Flags().String("host", "0.0.0.0", "Host address to listen on")
	serverCmd.Flags().Int("port", 8089, "Port to listen on")
	serverCmd.Flags().String("dockpilot-url", "http://127.0.0.1:8088", "Base URL of the dockpilot API server")
	rootCmd.AddCommand(serverCmd)
}
