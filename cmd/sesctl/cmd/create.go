package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func init() {
	cmdCreate := &cobra.Command{
		Use:              "create JOB_NAME [FLAGS]",
		Short:            "Create a job for submission",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			createFunc := func(r *JobRequest) error {
				if r.FilePath != "" {
					resp, err := r.handler.RequestPostFromFile("api/v1/create", r.FilePath)
					if err != nil {
						return err
					}
					body, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					blob, _ := json.MarshalIndent(body, "", " ")
					fmt.Printf("%s\n", string(blob))
				} else {
					if len(args) < 1 {
						return fmt.Errorf("Please specify job name")
					}
					name := args[0]
					q, err := url.ParseQuery("name=" + name)
					if err != nil {
						return err
					}
					resp, err := r.handler.RequestGet("api/v1/create", q, r.Headers)
					if err != nil {
						return err
					}
					body, err := r.handler.ParseJSONHTTPResponse(resp)
					if err != nil {
						return err
					}
					fmt.Printf("%v", body)
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
