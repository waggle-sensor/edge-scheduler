package cmd

import (
	"fmt"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
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
