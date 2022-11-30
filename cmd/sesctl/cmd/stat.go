package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
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
					var body map[string]interface{}
					decoder.Decode(&body)
					var (
						maxLengthID     int = 5
						maxLengthName   int = 26
						maxLengthUser   int = 8
						maxLengthStatus int = len("complete")
						maxAge          int = 5
						// maxLengthStartTime int = len("mm/dd/yyyy hh:MM:ss")
						// maxLengthDuration  int = len("mm/dd/yyyy hh:MM:ss")
					)
					formattedList := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, "JOB_ID", maxLengthName+3, "NAME", maxLengthUser+3, "USER", maxLengthStatus+3, "STATUS", maxAge+3, "AGE")
					formattedList += strings.Repeat("=", len(formattedList)) + "\n"
					for _, blob := range body {
						jobBlob, err := json.Marshal(blob)
						if err != nil {
							return err
						}
						var job *datatype.Job
						err = json.Unmarshal(jobBlob, &job)
						if err != nil {
							return err
						}
						if !showAll {
							if job.State.GetState() == datatype.JobRemoved || job.State.GetState() == datatype.JobComplete {
								continue
							}
						}
						var name string
						if len(job.Name) > maxLengthName {
							name = job.Name[:maxLengthName-1] + "..."
						} else {
							name = job.Name
						}
						switch job.State.GetState() {
						case datatype.JobRunning, datatype.JobComplete:
							t := time.Now().UTC()
							age := t.Sub(job.State.LastUpdated.Time).Round(1 * time.Second)
							formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.State.GetState(), maxAge, age)
							break
						default:
							formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.State.GetState(), maxAge, "-")
						}
					}
					fmt.Printf("%s", formattedList)
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
