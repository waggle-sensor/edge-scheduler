package cmd

import (
	"fmt"
	"net/url"
	"path"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
)

func init() {
	cmdSubmit := &cobra.Command{
		Use:              "submit [FLAGS]",
		Short:            "submit a job to cloud scheduler",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			submitFunc := func(r *JobRequest) error {
				subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_SUBMIT)
				if r.JobID != "" {
					q, err := url.ParseQuery("id=" + r.JobID + "&dryrun=" + fmt.Sprint(r.DryRun) + "&override=" + fmt.Sprint(r.Override))
					if err != nil {
						return err
					}
					resp, err := r.handler.RequestGet(subPathString, q, r.Headers)
					if err != nil {
						return err
					}
					decoder, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					fmt.Println(printSingleJsonFromDecoder(decoder))
				} else if r.FilePath != "" {
					q, err := url.ParseQuery("&dryrun=" + fmt.Sprint(r.DryRun))
					if err != nil {
						return err
					}
					resp, err := r.handler.RequestPostFromFile(subPathString, r.FilePath, q, r.Headers)
					if err != nil {
						return err
					}
					decoder, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					fmt.Println(printSingleJsonFromDecoder(decoder))
				} else {
					return fmt.Errorf("Either --job-id or --file-path should be provided.")
				}
				return nil
			}
			return jobRequest.Run(submitFunc)
		},
	}
	flags := cmdSubmit.Flags()
	flags.StringVarP(&jobRequest.JobID, "job-id", "j", "", "Job ID")
	flags.BoolVarP(&jobRequest.DryRun, "dry-run", "", false, "Dry run the job")
	flags.StringVarP(&jobRequest.FilePath, "file-path", "f", "", "Path to the job file")
	flags.BoolVar(&jobRequest.Override, "override", false, "Attempt to override the permission")
	rootCmd.AddCommand(cmdSubmit)
}
