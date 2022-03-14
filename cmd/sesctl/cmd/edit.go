package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	// TODO: edit does not yet support inline editing of jobs like "kubectl edit"
	var (
		filePath string
	)
	cmdEdit := &cobra.Command{
		Use:              "edit [FLAGS]",
		Short:            "Modify a job",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			if filePath != "" {
				resp, err := r.RequestPostFromFile("api/v1/edit", filePath)
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
				return fmt.Errorf("Interactive job editing is not supported.")
			}
			return nil
		},
	}
	flags := cmdEdit.Flags()
	flags.StringVarP(&filePath, "file-path", "f", "", "Path to the job file")
	rootCmd.AddCommand(cmdEdit)
}
