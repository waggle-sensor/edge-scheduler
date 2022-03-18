package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	var (
		jobID    string
		filePath string
		dryRun   bool
	)
	cmdSubmit := &cobra.Command{
		Use:              "submit [FLAGS]",
		Short:            "submit a job to cloud scheduler",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := interfacing.NewHTTPRequest(serverHostString)
			if jobID != "" {
				q, err := url.ParseQuery("id=" + jobID + "&dryrun=" + fmt.Sprint(dryRun))
				if err != nil {
					return err
				}
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
			} else if filePath != "" {
				q, err := url.ParseQuery("&dryrun=" + fmt.Sprint(dryRun))
				if err != nil {
					return err
				}
				resp, err := r.RequestPostFromFileWithQueries("api/v1/submit", filePath, q)
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
				return fmt.Errorf("Either --job-id or --file-path should be provided.")
			}
			return nil
		},
	}
	flags := cmdSubmit.Flags()
	flags.StringVarP(&jobID, "job-id", "j", "", "Job ID")
	flags.BoolVarP(&dryRun, "dry-run", "", false, "Dry run the job")
	flags.StringVarP(&filePath, "file-path", "f", "", "Path to the job file")
	rootCmd.AddCommand(cmdSubmit)
}
