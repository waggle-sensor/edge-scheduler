package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var (
		plugins      []string
		nodeSelector []string
		nodeVSN      []string
		output       string
	)
	cmdStat := &cobra.Command{
		Use:              "stat",
		Short:            "List jobs and science goals",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := RequestHTTP("http://localhost:9770/api/v1/jobs")
			if err != nil {
				return err
			}
			_, err = ParseJSONHTTPResponse(resp)
			if err != nil {
				return err
			}
			// PrintJobsPretty(body)
			fmt.Printf("")
			return nil
		},
	}
	flags := cmdStat.Flags()
	flags.StringSliceVarP(&plugins, "plugin", "p", []string{}, "Plugin Docker image and version")
	flags.StringSliceVarP(&nodeSelector, "node-selector", "s", []string{}, "Query string to select nodes")
	flags.StringSliceVarP(&nodeVSN, "vsn", "n", []string{}, "Node VSN name")
	flags.StringVarP(&output, "output", "o", "yaml", "Output type either of yaml or json")
	rootCmd.AddCommand(cmdStat)
}
