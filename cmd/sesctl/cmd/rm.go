package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	cmdRm := &cobra.Command{
		Use:              "rm [FLAGS] JOB_ID",
		Short:            "Remove or suspend a job",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobRequest.JobID = args[0]
			rmFunc := func(r *JobRequest) error {
				q, err := url.ParseQuery("id=" + r.JobID + "&suspend=" + strconv.FormatBool(r.Suspend) + "&force=" + strconv.FormatBool(r.Force))
				if err != nil {
					return err
				}
				resp, err := r.handler.RequestGet(fmt.Sprintf("api/v1/jobs/%s/rm", r.JobID), q, r.Headers)
				if err != nil {
					return err
				}
				body, err := r.handler.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				blob, _ := json.MarshalIndent(body, "", " ")
				fmt.Printf("%s\n", string(blob))
				return nil
			}
			return jobRequest.Run(rmFunc)
		},
	}
	flags := cmdRm.Flags()
	flags.BoolVarP(&jobRequest.Suspend, "suspend", "s", false, "Suspend the job")
	flags.BoolVarP(&jobRequest.Force, "force", "f", false, "Remove or suspend the job forcefully")
	rootCmd.AddCommand(cmdRm)
}
