package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"stackpilot/internal/dockpilot"
	"stackpilot/internal/stack"
	"stackpilot/internal/utils"
)

var stackRemoveVolumesFlag bool

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage multi-service stacks defined in YAML files",
}

var stackDeployCmd = &cobra.Command{
	Use:   "deploy <stack.yaml>",
	Short: "Deploy all services defined in a stack file",
	Args:  cobra.ExactArgs(1),
	RunE:  runStackDeploy,
}

var stackRemoveCmd = &cobra.Command{
	Use:   "remove <stack.yaml>",
	Short: "Stop and remove all services defined in a stack file",
	Args:  cobra.ExactArgs(1),
	RunE:  runStackRemove,
}

var stackStatusCmd = &cobra.Command{
	Use:   "status <stack.yaml>",
	Short: "Show runtime status of all services in a stack",
	Args:  cobra.ExactArgs(1),
	RunE:  runStackStatus,
}

var stackValidateCmd = &cobra.Command{
	Use:   "validate <stack.yaml>",
	Short: "Validate a stack file without deploying anything",
	Args:  cobra.ExactArgs(1),
	RunE:  runStackValidate,
}

func init() {
	stackRemoveCmd.Flags().BoolVarP(&stackRemoveVolumesFlag, "volumes", "v", false,
		"Also remove named volumes declared in the stack file (data will be lost)")

	stackCmd.PersistentFlags().StringVar(&dockpilotURL, "dockpilot-url", "http://127.0.0.1:8088",
		"Base URL of the dockpilot API server")

	stackCmd.AddCommand(stackDeployCmd, stackRemoveCmd, stackStatusCmd, stackValidateCmd)
	rootCmd.AddCommand(stackCmd)
}

func parseAndValidate(path string) (*stack.Stack, error) {
	s, err := stack.Parse(path)
	if err != nil {
		return nil, err
	}
	return s, stack.Validate(s)
}

func newClient() *dockpilot.Client {
	return dockpilot.New(dockpilotURL)
}

func runStackDeploy(cmd *cobra.Command, args []string) error {
	s, err := parseAndValidate(args[0])
	if err != nil {
		return err
	}

	client := newClient()
	if err := client.Health(cmd.Context()); err != nil {
		return fmt.Errorf("cannot reach dockpilot at %s: %w", dockpilotURL, err)
	}

	utils.PrintInfo(fmt.Sprintf("Deploying stack %q (%d service(s)) via %s",
		s.Name, len(s.Services), dockpilotURL))
	fmt.Println()

	if err := stack.Deploy(cmd.Context(), client, s); err != nil {
		return err
	}

	fmt.Println()
	utils.PrintSuccess(fmt.Sprintf("Stack %q deployed", s.Name))
	return nil
}

func runStackRemove(cmd *cobra.Command, args []string) error {
	s, err := parseAndValidate(args[0])
	if err != nil {
		return err
	}

	client := newClient()
	if err := client.Health(cmd.Context()); err != nil {
		return fmt.Errorf("cannot reach dockpilot at %s: %w", dockpilotURL, err)
	}

	utils.PrintInfo(fmt.Sprintf("Removing stack %q via %s", s.Name, dockpilotURL))

	if err := stack.Remove(cmd.Context(), client, s, stackRemoveVolumesFlag); err != nil {
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Stack %q removed", s.Name))
	return nil
}

func runStackStatus(cmd *cobra.Command, args []string) error {
	s, err := parseAndValidate(args[0])
	if err != nil {
		return err
	}

	client := newClient()
	statuses, err := stack.Status(cmd.Context(), client, s)
	if err != nil {
		return err
	}

	fmt.Printf("Stack: %s\n\n", s.Name)

	w := utils.NewTabWriter(os.Stdout)
	fmt.Fprintln(w, "SERVICE\tCONTAINER\tSTATE\tPORTS")
	fmt.Fprintln(w, "-------\t---------\t-----\t-----")
	for i, ns := range s.Services {
		st := statuses[i]
		ports := st.Ports
		if ports == "" {
			ports = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ns.Key, st.Container, st.State, ports)
	}
	return w.Flush()
}

func runStackValidate(_ *cobra.Command, args []string) error {
	s, err := parseAndValidate(args[0])
	if err != nil {
		return err
	}

	ordered, err := stack.ResolveOrder(s.Services)
	if err != nil {
		return err
	}

	utils.PrintSuccess(fmt.Sprintf("Stack %q is valid (%d service(s))", s.Name, len(s.Services)))
	for _, ns := range s.Services {
		fmt.Printf("  %-20s  image=%-30s  container=%s\n",
			ns.Key, ns.Def.Image, ns.Def.ContainerName)
	}

	fmt.Println()
	utils.PrintInfo("Deployment order:")
	for i, ns := range ordered {
		dep := ""
		if len(ns.Def.DependsOn) > 0 {
			dep = fmt.Sprintf("  (depends on: %s)", strings.Join(ns.Def.DependsOn, ", "))
		}
		fmt.Printf("  %d. %s%s\n", i+1, ns.Key, dep)
	}
	return nil
}
