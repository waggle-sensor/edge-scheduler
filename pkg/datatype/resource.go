package resource

// Resource structs resources used in scheduling
type Resource struct {
	CPU       int `yaml:"cpu,omitempty"`
	Memory    int `yaml:"memory,omitempty"`
	GPUMemory int `yaml:"gpumemory,omitempty"`
}
