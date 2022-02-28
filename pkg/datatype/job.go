package datatype

import (
	"bytes"
	"encoding/json"
)

// Job structs user request for jobs
type Job struct {
	Name               string   `json:"name,omitempty"`
	PluginTags         []string `json:"plugin_tags,omitempty"`
	Plugins            []string `json:"plugins,omitempty"`
	NodeTags           []string `json:"node_tags,omitempty"`
	Nodes              []string `json:"nodes,omitempty"`
	MinimumPerformance []string `json:"minimum_performance,omitempty"`
	ScienceRules       []string `json:"science_rules,omitempty"`
	SuccessCriteria    []string `json:"success_criteria,omitempty"`
}

// EncodeToJson returns encoded json of the job.
func (j *Job) EncodeToJson() ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(bf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", " ")
	err := encoder.Encode(j)
	return bf.Bytes(), err
}
