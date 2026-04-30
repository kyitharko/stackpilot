package stack

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"stackpilot/internal/dockpilot"
	"stackpilot/internal/utils"
)

// mergeEnv combines an env list with an environment map into a single slice.
func mergeEnv(list []string, envMap map[string]string) []string {
	if len(envMap) == 0 {
		return list
	}
	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	merged := make([]string, len(list), len(list)+len(keys))
	copy(merged, list)
	for _, k := range keys {
		merged = append(merged, k+"="+envMap[k])
	}
	return merged
}

// Deploy calls the dockpilot API to deploy all services in dependency order.
// Services whose container already exists on dockpilot are skipped.
func Deploy(ctx context.Context, client *dockpilot.Client, s *Stack) error {
	ordered, err := ResolveOrder(s.Services)
	if err != nil {
		return err
	}

	names := make([]string, len(ordered))
	for i, ns := range ordered {
		names[i] = ns.Key
	}
	utils.PrintInfo("Deployment order: " + strings.Join(names, " → "))
	fmt.Println()

	for _, ns := range ordered {
		req := dockpilot.DeployRequest{
			Image:   ns.Def.Image,
			Ports:   ns.Def.Ports,
			Volumes: ns.Def.Volumes,
			Env:     mergeEnv(ns.Def.Env, ns.Def.Environment),
			Command: ns.Def.Command,
		}

		utils.PrintInfo(fmt.Sprintf("[%s] Deploying container %q...", ns.Key, ns.Def.ContainerName))
		result, err := client.Deploy(ctx, ns.Def.ContainerName, req)
		if err != nil {
			// Treat "already exists" as a skip, matching the CLI behaviour.
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "CONFLICT") {
				utils.PrintWarning(fmt.Sprintf("[%s] Already deployed — skipping", ns.Key))
				continue
			}
			return fmt.Errorf("[%s] deploy failed: %w", ns.Key, err)
		}
		utils.PrintSuccess(fmt.Sprintf("[%s] Deployed → %s", ns.Key, result.Container))
	}
	return nil
}

// Remove calls the dockpilot API to remove all services in reverse dependency order.
// If removeVolumes is true, named volumes declared in the YAML are also deleted.
// Missing containers are skipped with a warning.
func Remove(ctx context.Context, client *dockpilot.Client, s *Stack, removeVolumes bool) error {
	ordered, err := ResolveOrder(s.Services)
	if err != nil {
		return err
	}

	// Reverse: teardown dependents before their dependencies.
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}

	for _, ns := range ordered {
		var volumes []string
		if removeVolumes {
			for _, vol := range ns.Def.Volumes {
				volumes = append(volumes, strings.SplitN(vol, ":", 2)[0])
			}
		}

		utils.PrintInfo(fmt.Sprintf("[%s] Removing container %q...", ns.Key, ns.Def.ContainerName))
		if err := client.Remove(ctx, ns.Def.ContainerName, volumes); err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NOT_FOUND") {
				utils.PrintWarning(fmt.Sprintf("[%s] Not deployed — skipping", ns.Key))
				continue
			}
			return fmt.Errorf("[%s] remove failed: %w", ns.Key, err)
		}
		utils.PrintSuccess(fmt.Sprintf("[%s] Removed %q", ns.Key, ns.Def.ContainerName))
	}
	return nil
}

// Status prints the runtime state of every service by querying the dockpilot API.
// Returns a slice of per-service status (also printed to stdout as a table).
func Status(ctx context.Context, client *dockpilot.Client, s *Stack) ([]dockpilot.ServiceStatus, error) {
	results := make([]dockpilot.ServiceStatus, 0, len(s.Services))

	for _, ns := range s.Services {
		status, err := client.Status(ctx, ns.Def.ContainerName)
		if err != nil {
			// Non-fatal: show as "unknown" so other services still show.
			utils.PrintWarning(fmt.Sprintf("[%s] Could not fetch status: %v", ns.Key, err))
			results = append(results, dockpilot.ServiceStatus{
				Name:      ns.Key,
				Container: ns.Def.ContainerName,
				State:     "unknown",
			})
			continue
		}
		results = append(results, *status)
	}
	return results, nil
}
