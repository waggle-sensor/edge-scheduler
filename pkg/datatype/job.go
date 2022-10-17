package datatype

import (
	"bytes"
	"encoding/json"
	"time"

	"gopkg.in/yaml.v2"
)

type JobState string

const (
	JobCreated   JobState = "Created"
	JobDrafted   JobState = "Drafted"
	JobSubmitted JobState = "Submitted"
	JobRunning   JobState = "Running"
	JobComplete  JobState = "Completed"
	JobSuspended JobState = "Suspended"
	JobRemoved   JobState = "Removed"
)

type Time struct {
	time.Time
}

// MarshalJSON overrides the JSON marshal function to send out null
// instead of skipping the attribute in JSON marshalling
func (t *Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return json.Marshal(nil)
	} else {
		return json.Marshal(t.Time)
	}
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var a time.Time
	err := json.Unmarshal(data, &a)
	if err != nil {
		return err
	}
	if !a.IsZero() {
		t.Time = a
	}
	return nil
}

type State struct {
	LastState     JobState `json:"last_state" yaml:"lastState"`
	LastUpdated   Time     `json:"last_updated" yaml:"lastUpdated"`
	LastSubmitted Time     `json:"last_submitted" yaml:"lastSubmitted"`
	LastStarted   Time     `json:"last_started" yaml:"lastStarted"`
	LastCompleted Time     `json:"last_completed" yaml:"lastCompleted"`
}

func (s *State) GetState() JobState {
	return s.LastState
}

func (s *State) UpdateState(newState JobState) {
	s.LastState = newState
	s.LastUpdated.Time = time.Now().UTC()
	switch newState {
	case JobSubmitted:
		s.LastSubmitted.Time = s.LastUpdated.Time
	case JobRunning:
		s.LastStarted.Time = s.LastUpdated.Time
	case JobComplete, JobSuspended, JobRemoved:
		s.LastCompleted.Time = s.LastUpdated.Time
	}
}

// Job structs user request for jobs
type Job struct {
	Name            string                 `json:"name" yaml:"name"`
	JobID           string                 `json:"job_id" yaml:"jobID"`
	User            string                 `json:"user" yaml:"user"`
	Email           string                 `json:"email" yaml:"email"`
	NotificationOn  []JobState             `json:"notification_on" yaml:"notificationOn"`
	Plugins         []*Plugin              `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	NodeTags        []string               `json:"node_tags" yaml:"nodeTags"`
	Nodes           map[string]interface{} `json:"nodes" yaml:"nodes"`
	ScienceRules    []string               `json:"science_rules" yaml:"scienceRules"`
	SuccessCriteria []string               `json:"success_criteria" yaml:"successCriteria"`
	ScienceGoal     *ScienceGoal           `json:"science_goal,omitempty" yaml:"scienceGoal,omitempty"`
	State           State                  `json:"state" yaml:"state"`
}

func NewJob(name string, user string, jobID string) *Job {
	return &Job{
		Name:  name,
		JobID: jobID,
		User:  user,
		Nodes: make(map[string]interface{}),
	}
}

func (j *Job) SetNotification(email string, on []JobState) {
	j.Email = email
	j.NotificationOn = on
}

func (j *Job) UpdateState(newState JobState) {
	j.State.UpdateState(newState)
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
