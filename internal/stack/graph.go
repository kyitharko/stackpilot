package stack

import (
	"fmt"
	"sort"
	"strings"
)

// GraphError is returned when the dependency graph contains a cycle.
type GraphError struct {
	CycleMembers []string
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("circular dependency detected involving: %s",
		strings.Join(e.CycleMembers, ", "))
}

// ResolveOrder returns services in topological (dependency-first) order using
// Kahn's algorithm. Among services with equal priority, YAML document order is
// preserved. Returns *GraphError when a cycle exists.
func ResolveOrder(services []NamedService) ([]NamedService, error) {
	n := len(services)
	if n == 0 {
		return nil, nil
	}

	idx := make(map[string]int, n)
	for i, ns := range services {
		idx[ns.Key] = i
	}

	inDegree := make([]int, n)
	dependents := make([][]int, n)

	for i, ns := range services {
		for _, dep := range ns.Def.DependsOn {
			depIdx, ok := idx[dep]
			if !ok {
				continue
			}
			inDegree[i]++
			dependents[depIdx] = append(dependents[depIdx], i)
		}
	}

	queue := make([]int, 0, n)
	for i, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]NamedService, 0, n)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, services[cur])

		deps := make([]int, len(dependents[cur]))
		copy(deps, dependents[cur])
		sort.Ints(deps)

		for _, d := range deps {
			inDegree[d]--
			if inDegree[d] == 0 {
				queue = append(queue, d)
			}
		}
	}

	if len(result) != n {
		var stuck []string
		for i, deg := range inDegree {
			if deg > 0 {
				stuck = append(stuck, services[i].Key)
			}
		}
		sort.Strings(stuck)
		return nil, &GraphError{CycleMembers: stuck}
	}
	return result, nil
}
