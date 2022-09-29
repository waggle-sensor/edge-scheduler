package cmd

import (
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
				decoder, err := r.handler.ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				fmt.Println(printSingleJsonFromDecoder(decoder))
				return nil
			}
			return jobRequest.Run(pingFunc)
		},
	}
	rootCmd.AddCommand(cmdPing)
}
