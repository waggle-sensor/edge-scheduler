package cmd

import (
	"fmt"
	"strings"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/spf13/cobra"
)

func init() {
	var (
		plugins      []string
		nodeSelector []string
		nodeVSN      []string
		output       string
	)
	cmdCreate := &cobra.Command{
		Use:              "create [FLAGS] JOB_NAME",
		Short:            "Create a job template for submission",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if len(plugins) < 1 {
				return fmt.Errorf("No plugin is selected. Please specify at least one plugin.")
			}
			job := &datatype.Job{
				Name:            name,
				NodeTags:        nodeSelector,
				Nodes:           nodeVSN,
				ScienceRules:    []string{"#Please specify science rules"},
				SuccessCriteria: []string{"WallClock(7d)"},
			}
			switch strings.ToLower(output) {
			case "yaml":
				blob, _ := job.EncodeToYaml()
				fmt.Printf("%s", string(blob))
				break
			case "json":
				blob, _ := job.EncodeToJson()
				fmt.Printf("%s", string(blob))
				break
			default:
				return fmt.Errorf("Unrecognized output: %q", output)
			}
			return nil
		},
	}
	flags := cmdCreate.Flags()
	flags.StringSliceVarP(&plugins, "plugin", "p", []string{}, "Plugin Docker image and version")
	flags.StringSliceVarP(&nodeSelector, "node-selector", "s", []string{}, "Query string to select nodes")
	flags.StringSliceVarP(&nodeVSN, "vsn", "n", []string{}, "Node VSN name")
	flags.StringVarP(&output, "output", "o", "yaml", "Output type either of yaml or json")
	rootCmd.AddCommand(cmdCreate)
}
