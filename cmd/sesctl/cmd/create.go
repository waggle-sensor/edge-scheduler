package cmd

import (
	"fmt"
	"net/url"
	"path"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
)

func init() {
	cmdCreate := &cobra.Command{
		Use:              "create JOB_NAME [FLAGS]",
		Short:            "Create a job for submission",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			createFunc := func(r *JobRequest) error {
				subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_CREATE)
				if r.FilePath != "" {
					resp, err := r.handler.RequestPostFromFile(subPathString, r.FilePath, nil, r.Headers)
					if err != nil {
						return err
					}
					decoder, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					fmt.Println(printSingleJsonFromDecoder(decoder))
				} else {
					if len(args) < 1 {
						return fmt.Errorf("Please specify job name")
					}
					name := args[0]
					q, err := url.ParseQuery("name=" + name)
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
				}
				return nil
			}
			return jobRequest.Run(createFunc)
		},
	}
	flags := cmdCreate.Flags()
	flags.StringVarP(&jobRequest.FilePath, "file-path", "f", "", "Path to job file")
	rootCmd.AddCommand(cmdCreate)
}
