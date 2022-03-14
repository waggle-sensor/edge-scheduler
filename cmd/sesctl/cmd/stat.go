package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	var (
		jobName string
		outPath string
	)
	cmdStat := &cobra.Command{
		Use:              "stat [FLAGS]",
		Short:            "List jobs",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			if jobName != "" {
				resp, err := r.RequestGet(fmt.Sprintf("api/v1/jobs/%s/status", jobName), nil)
				if err != nil {
					return err
				}
				body, err := r.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				if blob, exist := body[jobName]; exist {
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
					maxLengthName      int = 5
					maxLengthStatus    int = len("succeeded")
					maxLengthStartTime int = 35
					maxLengthDuration  int = 4
				)
				formattedList := fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, "NAME", maxLengthStatus+3, "STATUS", maxLengthStartTime+3, "START_TIME", maxLengthDuration+3, "RUNNING_TIME")
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
					switch job.Status {
					case datatype.JobRunning, datatype.JobComplete:
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, job.Name, maxLengthStatus+3, job.Status, maxLengthStartTime+3, job.LastUpdated, maxLengthDuration+3, time.Since(job.LastUpdated))
						break
					default:
						formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, job.Name, maxLengthStatus+3, job.Status, maxLengthStartTime+3, job.LastUpdated, maxLengthDuration+3, "")
					}
				}
				fmt.Printf("%s", formattedList)
			}
			return nil
		},
	}
	flags := cmdStat.Flags()
	flags.StringVarP(&jobName, "job", "j", "", "Name of the job")
	flags.StringVarP(&outPath, "out", "o", "", "Path to save output")
	rootCmd.AddCommand(cmdStat)
}
