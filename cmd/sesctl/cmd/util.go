package cmd

import (
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
Job Starttime: %s
`,
		j.JobID,
		j.Name,
		j.User,
		j.Status,
		j.LastUpdated.UTC().String(),
	)
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
