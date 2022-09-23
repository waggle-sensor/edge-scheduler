package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/cloudscheduler"
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
				subPathString := path.Join(cloudscheduler.API_V1_VERSION, cloudscheduler.API_PATH_JOB_EDIT)
				if r.FilePath != "" {
					q, err := url.ParseQuery("&id=" + fmt.Sprint(args[0]))
					if err != nil {
						return err
					}
					resp, err := r.handler.RequestPostFromFile(subPathString, r.FilePath, q, nil)
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
