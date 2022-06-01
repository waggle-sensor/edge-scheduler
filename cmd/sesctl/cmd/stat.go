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
						fmt.Printf("%s\n", jobBlob)
					}
				} else {
					fmt.Printf("%v\n", body)
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
					maxLengthID        int = 5
					maxLengthName      int = 6
					maxLengthUser      int = 8
					maxLengthStatus    int = len("complete")
					maxLengthStartTime int = len("mm/dd/yyyy hh:MM:ss")
					maxLengthDuration  int = len("mm/dd/yyyy hh:MM:ss")
				)
				formattedList := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, "JOB_ID", maxLengthName+3, "NAME", maxLengthUser+3, "USER", maxLengthStatus+3, "STATUS", maxLengthStartTime+3, "START_TIME", maxLengthDuration+3, "RUNNING_TIME")
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
					var name string
					if len(job.Name) > maxLengthName {
						name = job.Name[:maxLengthName-1] + "..."
					} else {
						name = job.Name
					}
					switch job.Status {
					case datatype.JobSubmitted, datatype.JobRunning, datatype.JobComplete:
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.Status, maxLengthStartTime+3, job.LastUpdated.Format("01/02/2006 15:04:05"), maxLengthDuration+3, time.Since(job.LastUpdated))
						break
					default:
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s\n", maxLengthID+3, job.JobID, maxLengthName+3, name, maxLengthUser+3, job.User, maxLengthStatus+3, job.Status, maxLengthStartTime+3, "-", maxLengthDuration+3, "-")
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
	rootCmd.AddCommand(cmdStat)
}
