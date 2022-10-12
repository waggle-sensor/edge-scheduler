package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

func printJob(j *datatype.Job) (ret string) {
	ret = fmt.Sprintf(`
===== JOB STATUS =====
Job ID: %s
Job Name: %s
Job Owner: %s
Job Status: %s
  Last updated: %s
`,
		j.JobID,
		j.Name,
		j.User,
		j.State.GetState(),
		j.State.LastUpdated.Time.UTC().String(),
	)
	if !j.State.LastSubmitted.Time.IsZero() {
		ret += fmt.Sprintf(`  Submitted: %s
`,
			j.State.LastSubmitted)
	}
	if !j.State.LastStarted.Time.IsZero() {
		ret += fmt.Sprintf(`  Started: %s
`,
			j.State.LastStarted)
	}
	if !j.State.LastCompleted.Time.IsZero() {
		ret += fmt.Sprintf(`  Completed: %s
`,
			j.State.LastCompleted)
	}
	if len(j.NotificationOn) > 0 {
		ret += fmt.Sprintf(`
Notification to %s
On %v
`,
			j.Email, j.NotificationOn)
	}
	if j.ScienceGoal != nil {
		ret += fmt.Sprintf(`
===== SCHEDULING DETAILS =====
Science Goal ID: %s
Total number of nodes %d
`,
			j.ScienceGoal.ID,
			len(j.ScienceGoal.SubGoals),
		)
		// 		for _, subGoal := range j.ScienceGoal.SubGoals {
		// 			ret += fmt.Sprintf(`
		// Node %q:
		// `,
		// 				subGoal.Name,
		// 				len(subGoal.Plugins),
		// 			)
		// 		}
	}
	// 	ret += fmt.Sprintf(`
	// ===== SUBMITTED JOB INPUTS =====
	// NodeTag: %v
	// Nodes: %v
	// `,
	// 	j.NodeTags,
	// 	j.Nodes,
	// 	j.Plugins

	// )
	return ret
}

func printSingleJsonFromDecoder(decoder *json.Decoder) string {
	var blob map[string]interface{}
	decoder.Decode(&blob)
	ret, _ := json.MarshalIndent(blob, "", " ")
	return string(ret)
}

type JobRequest struct {
	ServerHostString string
	handler          *interfacing.HTTPRequest
	UserToken        string
	JobID            string
	OutPath          string            // for saving response into a file
	DryRun           bool              // dry-run of job submission
	FilePath         string            // for loading job description from a file
	Suspend          bool              // for suspending a job
	Force            bool              // for making the request forceful
	Headers          map[string]string // additional headers for request
}

func (r *JobRequest) open() {
	r.handler = interfacing.NewHTTPRequest(r.ServerHostString)
}

func (r *JobRequest) Run(f func(*JobRequest) error) error {
	if r.handler == nil {
		r.open()
	}
	// TODO: The token may use different name other than Sage
	r.Headers = map[string]string{
		"Accept":        "applications/json",
		"Authorization": fmt.Sprintf("Sage %s", r.UserToken),
	}
	return f(r)
}
