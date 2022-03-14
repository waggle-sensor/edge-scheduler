package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	cmdSubmit := &cobra.Command{
		Use:              "submit JOB_NAME",
		Short:            "submit a job to cloud scheduler",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobName := args[0]
			q, err := url.ParseQuery("name=" + jobName)
			if err != nil {
				return err
			}
			r := interfacing.NewHTTPRequest(serverHostString)
			resp, err := r.RequestGet("api/v1/submit", q)
			if err != nil {
				return err
			}
			body, err := r.ParseJSONHTTPResponse(resp)
			if err != nil {
				return err
			}
			blob, _ := json.MarshalIndent(body, "", " ")
			fmt.Printf("%s\n", string(blob))
			return nil
		},
	}
	rootCmd.AddCommand(cmdSubmit)
}
