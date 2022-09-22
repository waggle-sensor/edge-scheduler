package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	cmdPing := &cobra.Command{
		Use:              "ping",
		Short:            "Ping the Sage edge scheduler",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pingFunc := func(r *JobRequest) error {
				resp, err := r.handler.RequestGet("", nil, r.Headers)
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
			return jobRequest.Run(pingFunc)
		},
	}
	rootCmd.AddCommand(cmdPing)
}
