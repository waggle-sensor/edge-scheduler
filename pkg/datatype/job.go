package datatype

import (
	"bytes"
	"encoding/json"
	"time"

	"gopkg.in/yaml.v2"
)

type JobStatus string

const (
	JobCreated   JobStatus = "Created"
	JobDrafted   JobStatus = "Drafted"
	JobSubmitted JobStatus = "Submitted"
	JobRunning   JobStatus = "Running"
	JobComplete  JobStatus = "Complete"
	JobHalted    JobStatus = "Halted"
)

// Job structs user request for jobs
type Job struct {
	Name            string                 `json:"name" yaml:"name"`
	Plugins         []*Plugin              `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	NodeTags        []string               `json:"node_tags" yaml:"nodeTags"`
	Nodes           map[string]interface{} `json:"nodes" yaml:"nodes"`
	ScienceRules    []string               `json:"science_rules" yaml:"scienceRules"`
	SuccessCriteria []string               `json:"success_criteria" yaml:"successCriteria"`
	ScienceGoal     *ScienceGoal           `json:"science_goal,omitempty" yaml:"scienceGoal,omitempty"`
	Status          JobStatus              `json:"status" yaml:"status"`
	LastUpdated     time.Time              `json:"last_updated" yaml:"lastUpdated"`
}

func (j *Job) UpdateStatus(newStatus JobStatus) {
	j.Status = newStatus
	j.updateLastModified()
}

func (j *Job) AddNodes(nodeNames []string) {
	for _, nodeName := range nodeNames {
		if _, exist := j.Nodes[nodeName]; !exist {
			j.Nodes[nodeName] = 1
		}
	}
}

func (j *Job) DropNode(nodeName string) {
	if _, exist := j.Nodes[nodeName]; exist {
		delete(j.Nodes, nodeName)
	}
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

func (j *Job) updateLastModified() {
	j.LastUpdated = time.Now()
}
