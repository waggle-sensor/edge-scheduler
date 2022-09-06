package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

func init() {
	var (
		jobID   string
		outPath string
		showAll bool
	)
	cmdStat := &cobra.Command{
		Use:              "stat [FLAGS]",
		Short:            "List jobs",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			if jobID != "" {
				resp, err := r.RequestGet(fmt.Sprintf("api/v1/jobs/%s/status", jobID), nil)
				if err != nil {
					return err
				}
				body, err := r.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				if blob, exist := body[jobID]; exist {
					jobBlob, err := json.MarshalIndent(blob, "", "  ")
					if err != nil {
						return err
					}
					if outPath != "" {
						err := ioutil.WriteFile(outPath, jobBlob, 0644)
						if err != nil {
							return err
						}
					} else {
						var job datatype.Job
						err := json.Unmarshal(jobBlob, &job)
						if err != nil {
							return err
						}
						fmt.Print(printJob(&job))
					}
				} else {
					return fmt.Errorf("Failed to get the job %q: Job does not exist", jobID)
				}
			} else {
				resp, err := r.RequestGet("api/v1/jobs", nil)
				if err != nil {
					return err
				}
				body, err := r.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
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
						if job.Status == datatype.JobRemoved || job.Status == datatype.JobComplete {
							continue
						}
					}
					var name string
					if len(job.Name) > maxLengthName {
						name = job.Name[:maxLengthName-1] + "..."
					} else {
						name = job.Name
					}
					switch job.Status {
					case datatype.JobSubmitted, datatype.JobRunning, datatype.JobComplete:
						t := time.Now().UTC()
						age := t.Sub(job.LastUpdated).Round(1 * time.Second)
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.Status, maxAge, age)
						break
					default:
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.Status, maxAge, "-")
					}
				}
				fmt.Printf("%s", formattedList)
			}
			return nil
		},
	}
	flags := cmdStat.Flags()
	flags.StringVarP(&jobID, "job-id", "j", "", "Job ID")
	flags.StringVarP(&outPath, "out", "o", "", "Path to save output")
	flags.BoolVarP(&showAll, "show-all", "A", false, "Show all jobs including removed and completed jobs")
	rootCmd.AddCommand(cmdStat)
}
