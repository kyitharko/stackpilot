package stack

import (
	"errors"
	"testing"
)

func makeServices(keys []string, deps map[string][]string) []NamedService {
	result := make([]NamedService, len(keys))
	for i, k := range keys {
		result[i] = NamedService{
			Key: k,
			Def: ServiceDef{Image: k + ":latest", DependsOn: deps[k]},
		}
	}
	return result
}

func serviceKeys(services []NamedService) []string {
	out := make([]string, len(services))
	for i, ns := range services {
		out[i] = ns.Key
	}
	return out
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func indexOf(services []NamedService, key string) int {
	for i, ns := range services {
		if ns.Key == key {
			return i
		}
	}
	return -1
}

func TestResolveOrder_NoDeps_PreservesYAMLOrder(t *testing.T) {
	svc := makeServices([]string{"a", "b", "c"}, nil)
	got, err := ResolveOrder(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalSlice(serviceKeys(got), []string{"a", "b", "c"}) {
		t.Errorf("got %v, want [a b c]", serviceKeys(got))
	}
}

func TestResolveOrder_Empty(t *testing.T) {
	got, err := ResolveOrder(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestResolveOrder_LinearChain(t *testing.T) {
	svc := makeServices([]string{"c", "b", "a"}, map[string][]string{
		"c": {"b"}, "b": {"a"},
	})
	got, err := ResolveOrder(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalSlice(serviceKeys(got), []string{"a", "b", "c"}) {
		t.Errorf("got %v, want [a b c]", serviceKeys(got))
	}
}

func TestResolveOrder_Diamond_ApiLast(t *testing.T) {
	svc := makeServices([]string{"mongodb", "redis", "api"}, map[string][]string{
		"api": {"mongodb", "redis"},
	})
	got, err := ResolveOrder(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	apiIdx := indexOf(got, "api")
	if apiIdx != len(got)-1 {
		t.Errorf("api should be last, got %v", serviceKeys(got))
	}
	if indexOf(got, "mongodb") >= apiIdx || indexOf(got, "redis") >= apiIdx {
		t.Errorf("mongodb and redis must precede api, got %v", serviceKeys(got))
	}
}

func TestResolveOrder_TwoNodeCycle(t *testing.T) {
	svc := makeServices([]string{"a", "b"}, map[string][]string{
		"a": {"b"}, "b": {"a"},
	})
	_, err := ResolveOrder(svc)
	if err == nil {
		t.Fatal("expected *GraphError for two-node cycle, got nil")
	}
	var ge *GraphError
	if !errors.As(err, &ge) {
		t.Fatalf("expected *GraphError, got %T: %v", err, err)
	}
	if len(ge.CycleMembers) != 2 {
		t.Errorf("expected 2 cycle members, got %v", ge.CycleMembers)
	}
}

func TestResolveOrder_SelfDependency(t *testing.T) {
	svc := makeServices([]string{"a"}, map[string][]string{"a": {"a"}})
	_, err := ResolveOrder(svc)
	var ge *GraphError
	if !errors.As(err, &ge) {
		t.Fatalf("expected *GraphError, got %T: %v", err, err)
	}
}
