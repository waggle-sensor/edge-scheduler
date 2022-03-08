package datatype

import (
	"bytes"
	"encoding/json"

	"gopkg.in/yaml.v2"
)

// Job structs user request for jobs
type Job struct {
	Name            string       `json:"name,omitempty" yaml:"name,omitempty"`
	PluginTags      []string     `json:"plugin_tags,omitempty" yaml:"plugin_tags,omitempty"`
	Plugins         []PluginSpec `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	NodeTags        []string     `json:"node_tags,omitempty" yaml:"node_tags,omitempty"`
	Nodes           []string     `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	ScienceRules    []string     `json:"science_rules,omitempty" yaml:"science_rules,omitempty"`
	SuccessCriteria []string     `json:"success_criteria,omitempty" yaml:"success_criteria,omitempty"`
	ScienceGoal     *ScienceGoal
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

func (j *Job) EncodeToYaml() ([]byte, error) {
	return yaml.Marshal(j)
}
