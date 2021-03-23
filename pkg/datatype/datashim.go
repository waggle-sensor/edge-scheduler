package datatype

// DataShim structs a mapping of aliases used in plugin code to
// node resources available within the node
// DataShim is a json as PyWaggle requires a json type
type DataShim struct {
	Name    string          `yaml:"name,omitempty" json:"name,omitempty"`
	Match   DataShimMatch   `yaml:"match,omitempty" json:"match,omitempty"`
	Handler DataShimHandler `yaml:"handler,omitempty" json:"handler,omitempty"`
}

// DataShimMatch structs matching keys for data shim
type DataShimMatch struct {
	ID          string `yaml:"id,omitempty" json:"id,omitempty"`
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`
	Orientation string `yaml:"orientation,omitempty" json:"orientation,omitempty"`
	Resolution  string `yaml:"resolution,omitempty" json:"resolution,omitempty"`
}

// DataShimHandler defines node resource matching with matching keys
type DataShimHandler struct {
	Type string            `yaml:"type,omitempty" json:"type,omitempty"`
	Args map[string]string `yaml:"args,omitempty" json:"args,omitempty"`
}
