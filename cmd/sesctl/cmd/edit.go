package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

func init() {
	// TODO: edit does not yet support inline editing of jobs like "kubectl edit"
	var (
		filePath string
	)
	cmdEdit := &cobra.Command{
		Use:              "edit JOB_ID",
		Short:            "Modify an existing job",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("JOB_ID must be specified.")
			}
			r := interfacing.NewHTTPRequest(serverHostString)
			if filePath != "" {
				q, err := url.ParseQuery("&id=" + fmt.Sprint(args[0]))
				if err != nil {
					return err
				}
				resp, err := r.RequestPostFromFileWithQueries("api/v1/edit", filePath, q)
				if err != nil {
					return err
				}
				body, err := r.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				blob, _ := json.MarshalIndent(body, "", " ")
				fmt.Printf("%s\n", string(blob))
			} else {
				return fmt.Errorf("Interactive job editing is not supported. Please use -f to change job.")
			}
			return nil
		},
	}
	flags := cmdEdit.Flags()
	flags.StringVarP(&filePath, "file-path", "f", "", "Path to the job file")
	rootCmd.AddCommand(cmdEdit)
}
