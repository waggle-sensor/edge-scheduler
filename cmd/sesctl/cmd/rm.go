package cmd

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
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
				subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_REMOVE_REGEX)
				q, err := url.ParseQuery(
					"suspend=" + strconv.FormatBool(r.Suspend) +
						"&force=" + strconv.FormatBool(r.Force) +
						"&override=" + fmt.Sprint(r.Override))
				if err != nil {
					return err
				}
				resp, err := r.handler.RequestGet(fmt.Sprintf(subPathString, r.JobID), q, r.Headers)
				if err != nil {
					return err
				}
				decoder, err := r.handler.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				fmt.Println(printSingleJsonFromDecoder(decoder))
				return nil
			}
			return jobRequest.Run(rmFunc)
		},
	}
	flags := cmdRm.Flags()
	flags.BoolVarP(&jobRequest.Suspend, "suspend", "s", false, "Suspend the job")
	flags.BoolVarP(&jobRequest.Force, "force", "f", false, "Remove or suspend the job forcefully")
	flags.BoolVar(&jobRequest.Override, "override", false, "Attempt to override the permission")
	rootCmd.AddCommand(cmdRm)
}
