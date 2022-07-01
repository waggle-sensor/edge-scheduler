package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
)

func init() {
	var (
		filePath string
	)
	cmdCreate := &cobra.Command{
		Use:              "create JOB_NAME [FLAGS]",
		Short:            "Create a job for submission",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			if filePath != "" {
				resp, err := r.RequestPostFromFile("api/v1/create", filePath)
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
				if len(args) < 1 {
					return fmt.Errorf("Please specify job name")
				}
				name := args[0]
				q, err := url.ParseQuery("name=" + name)
				if err != nil {
					return err
				}
				resp, err := r.RequestGet("api/v1/create", q)
				if err != nil {
					return err
				}
				body, err := r.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				fmt.Printf("%v", body)
				// logger.Debug.Printf("%s", body["name"])
			}
			return nil
		},
	}
	flags := cmdCreate.Flags()
	flags.StringVarP(&filePath, "file-path", "f", "", "Path to job file")
	rootCmd.AddCommand(cmdCreate)
}
