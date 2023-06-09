package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

func init() {
	var showAll bool
	cmdStat := &cobra.Command{
		Use:              "stat [FLAGS]",
		Short:            "List jobs",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			statFunc := func(r *JobRequest) error {
				if r.JobID != "" {
					subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_STATUS_REGEX)
					resp, err := r.handler.RequestGet(fmt.Sprintf(subPathString, r.JobID), nil, r.Headers)
					if err != nil {
						return err
					}
					decoder, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					var job datatype.Job
					err = decoder.Decode(&job)
					if err != nil {
						return err
					}
					if r.OutPath != "" {
						jobBlob, err := json.MarshalIndent(job, "", "  ")
						if err != nil {
							return err
						}
						err = ioutil.WriteFile(r.OutPath, jobBlob, 0644)
						if err != nil {
							return err
						}
					} else {
						fmt.Print(printJob(&job))
					}
				} else {
					subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_LIST)
					resp, err := r.handler.RequestGet(subPathString, nil, r.Headers)
					if err != nil {
						return err
					}
					decoder, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}

					var jobs map[string]*datatype.Job
					if err := decoder.Decode(&jobs); err != nil {
						return err
					}

					writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

					fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", "JOB_ID", "NAME", "USER", "STATUS", "AGE")

					for _, job := range jobs {
						if !showAll && (job.State.GetState() == datatype.JobRemoved || job.State.GetState() == datatype.JobComplete) {
							continue
						}
						fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", job.JobID, job.Name, job.User, job.State.GetState(), getJobAgeString(job))
					}

					writer.Flush()
				}
				return nil
			}
			return jobRequest.Run(statFunc)
		},
	}
	flags := cmdStat.Flags()
	flags.StringVarP(&jobRequest.JobID, "job-id", "j", "", "Job ID")
	flags.StringVarP(&jobRequest.OutPath, "out", "o", "", "Path to save output")
	flags.BoolVarP(&showAll, "show-all", "A", false, "Show all jobs including removed and completed jobs")
	rootCmd.AddCommand(cmdStat)
}

func getJobAgeString(job *datatype.Job) string {
	switch {
	case job == nil:
		return "-"
	case job.State.LastState == datatype.JobRunning || job.State.LastState == datatype.JobComplete:
		return time.Since(job.State.LastUpdated.Time).Round(1 * time.Second).String()
	default:
		return "-"
	}
}
