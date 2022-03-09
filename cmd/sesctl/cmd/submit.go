package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	cmdSubmit := &cobra.Command{
		Use:              "submit JOB_FILE",
		Short:            "submit a job to cloud scheduler",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			job_filepath := args[0]
			if _, err := os.Stat(job_filepath); os.IsNotExist(err) {
				return fmt.Errorf("%s not exist", job_filepath)
			}
			// job := &datatype.Job{
			// 	Name:            name,
			// 	Plugins:         plugins,
			// 	NodeTags:        nodeSelector,
			// 	Nodes:           nodeVSN,
			// 	ScienceRules:    []string{"#Please specify science rules"},
			// 	SuccessCriteria: []string{"WallClockDays(7)"},
			// }
			// jsonblob, _ := job.EncodeToJson()
			// if len(output) > 0 {
			// 	err := os.MkdirAll(output, os.ModePerm)
			// 	if err != nil {
			// 		return err
			// 	}
			// 	err = ioutil.WriteFile(filepath.Join(output, fmt.Sprintf("%s.json", name)), jsonblob, 0644)
			// 	if err != nil {
			// 		return err
			// 	}
			// } else {
			// 	fmt.Printf("%s", string(jsonblob))
			// }
			return nil
		},
	}
	// flags := cmdSubmit.Flags()
	// flags.StringSliceVarP(&plugins, "plugin", "p", []string{}, "Plugin Docker image and version")
	// flags.StringSliceVarP(&nodeSelector, "node-selector", "s", []string{}, "Query string to select nodes")
	// flags.StringSliceVarP(&nodeVSN, "vsn", "n", []string{}, "Node VSN name")
	// flags.StringVarP(&output, "output", "o", "", "Output path of the job")
	rootCmd.AddCommand(cmdSubmit)
}
