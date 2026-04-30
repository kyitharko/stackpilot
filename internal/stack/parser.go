package stack

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse reads path, decodes it as a stack YAML, and returns an ordered Stack.
// Container names not specified in the file are auto-filled as <stackname>-<key>.
func Parse(path string) (*Stack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseBytes(data, path)
}

// ParseBytes decodes raw YAML bytes into an ordered Stack.
// source is used only in error messages (e.g. a file path or "<stdin>").
func ParseBytes(data []byte, source string) (*Stack, error) {
	var raw struct {
		Name     string    `yaml:"name"`
		Services yaml.Node `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", source, err)
	}

	s := &Stack{Name: raw.Name}

	node := raw.Services
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = *node.Content[0]
	}
	if node.Kind == 0 {
		return s, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: 'services' must be a YAML mapping", source)
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		var def ServiceDef
		if err := node.Content[i+1].Decode(&def); err != nil {
			return nil, fmt.Errorf("%s: parsing service %q: %w", source, key, err)
		}
		if def.ContainerName == "" {
			def.ContainerName = raw.Name + "-" + key
		}
		s.Services = append(s.Services, NamedService{Key: key, Def: def})
	}

	return s, nil
}
