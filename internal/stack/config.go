package stack

// ServiceDef is a single service entry from the stack YAML.
type ServiceDef struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name"`
	Ports         []string          `yaml:"ports"`
	Volumes       []string          `yaml:"volumes"`
	Env           []string          `yaml:"env"`
	DependsOn     []string          `yaml:"depends_on"`
	Environment   map[string]string `yaml:"environment"`
	Command       []string          `yaml:"command"`
}

// NamedService pairs the YAML key with its parsed definition.
type NamedService struct {
	Key string
	Def ServiceDef
}

// Stack is the validated, ordered in-memory representation of a stack file.
// Services are stored in YAML document order for deterministic deployment.
type Stack struct {
	Name     string
	Services []NamedService
}
