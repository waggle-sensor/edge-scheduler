package cmd

import (
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/spf13/cobra"
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
			name := args[0]
			if filePath != "" {
				// job := &datatype.Job{
				// 	Name:            name,
				// 	NodeTags:        nodeSelector,
				// 	Nodes:           nodeVSN,
				// 	ScienceRules:    []string{"#Please specify science rules"},
				// 	SuccessCriteria: []string{"WallClock(7d)"},
				// }
				// switch strings.ToLower(output) {
				// case "yaml":
				// 	blob, _ := job.EncodeToYaml()
				// 	fmt.Printf("%s", string(blob))
				// 	break
				// case "json":
				// 	blob, _ := job.EncodeToJson()
				// 	fmt.Printf("%s", string(blob))
				// 	break
				// }
			} else {
				resp, err := RequestHTTP("http://localhost:9770/api/v1/create?name=" + name)
				if err != nil {
					return err
				}
				body, err := ParseJSONHTTPResponse(resp)
				if err != nil {
					return err
				}
				logger.Debug.Printf("%v", body)
				// logger.Debug.Printf("%s", body["name"])
			}
			return nil
		},
	}
	flags := cmdCreate.Flags()
	flags.StringVarP(&filePath, "filepath", "f", "", "Path to job file")
	rootCmd.AddCommand(cmdCreate)
}
