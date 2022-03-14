package datatype

// Resource structs resources used in scheduling
type Resource struct {
	CPU       string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory    string `json:"memory,omitempty" yaml:"memory,omitempty"`
	GPUMemory string `json:"gpu_memory,omitempty" yaml:"gpuMemory,omitempty"`
}
