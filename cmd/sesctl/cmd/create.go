package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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
				Plugins:         plugins,
				NodeTags:        nodeSelector,
				Nodes:           nodeVSN,
				ScienceRules:    []string{"#Please specify science rules"},
				SuccessCriteria: []string{"WallClockDays(7)"},
			}
			jsonblob, _ := job.EncodeToJson()
			if len(output) > 0 {
				err := os.MkdirAll(output, os.ModePerm)
				if err != nil {
					return err
				}
				err = ioutil.WriteFile(filepath.Join(output, fmt.Sprintf("%s.json", name)), jsonblob, 0644)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("%s", string(jsonblob))
			}
			return nil
		},
	}
	flags := cmdCreate.Flags()
	flags.StringSliceVarP(&plugins, "plugin", "p", []string{}, "Plugin Docker image and version")
	flags.StringSliceVarP(&nodeSelector, "node-selector", "s", []string{}, "Query string to select nodes")
	flags.StringSliceVarP(&nodeVSN, "vsn", "n", []string{}, "Node VSN name")
	flags.StringVarP(&output, "output", "o", "", "Output path of the job")
	rootCmd.AddCommand(cmdCreate)
}
