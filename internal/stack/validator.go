package stack

import (
	"fmt"
	"strings"
)

// ValidationError collects every structural issue found so the user sees all
// problems at once rather than one error per run.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return "stack validation failed:\n  " + strings.Join(e.Errors, "\n  ")
}

// Validate checks a parsed Stack for required fields, duplicate names, and
// depends_on correctness. It does not detect cycles — ResolveOrder handles that.
func Validate(s *Stack) error {
	var errs []string

	if s.Name == "" {
		errs = append(errs, "missing required field 'name'")
	}
	if len(s.Services) == 0 {
		errs = append(errs, "no services defined under 'services'")
	}

	knownKeys := make(map[string]bool, len(s.Services))
	for _, ns := range s.Services {
		knownKeys[ns.Key] = true
	}

	seenContainers := map[string]bool{}
	for _, ns := range s.Services {
		prefix := fmt.Sprintf("service %q", ns.Key)

		if ns.Def.Image == "" {
			errs = append(errs, prefix+": missing required field 'image'")
		}
		if seenContainers[ns.Def.ContainerName] {
			errs = append(errs, prefix+fmt.Sprintf(": duplicate container_name %q", ns.Def.ContainerName))
		}
		seenContainers[ns.Def.ContainerName] = true

		for _, dep := range ns.Def.DependsOn {
			if dep == ns.Key {
				errs = append(errs, prefix+": depends_on references itself")
				continue
			}
			if !knownKeys[dep] {
				errs = append(errs, prefix+fmt.Sprintf(": depends_on references unknown service %q", dep))
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}
