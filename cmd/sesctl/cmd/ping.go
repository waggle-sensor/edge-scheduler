package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	cmdPing := &cobra.Command{
		Use:              "ping",
		Short:            "Ping the Sage edge scheduler",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			resp, err := r.RequestGet("", map[string][]string{})
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
	rootCmd.AddCommand(cmdPing)
}
