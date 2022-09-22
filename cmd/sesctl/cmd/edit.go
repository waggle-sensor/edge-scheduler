package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func init() {
	// TODO: edit does not yet support inline editing of jobs like "kubectl edit"
	cmdEdit := &cobra.Command{
		Use:              "edit JOB_ID",
		Short:            "Modify an existing job",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobRequest.JobID = args[0]
			editFunc := func(r *JobRequest) error {
				if r.FilePath != "" {
					q, err := url.ParseQuery("&id=" + fmt.Sprint(args[0]))
					if err != nil {
						return err
					}
					resp, err := r.handler.RequestPostFromFileWithQueries("api/v1/edit", r.FilePath, q)
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
					return fmt.Errorf("Interactive job editing is not supported. Please use -f to change job.")
				}
				return nil
			}
			return jobRequest.Run(editFunc)
		},
	}
	flags := cmdEdit.Flags()
	flags.StringVarP(&jobRequest.FilePath, "file-path", "f", "", "Path to the job file")
	rootCmd.AddCommand(cmdEdit)
}
