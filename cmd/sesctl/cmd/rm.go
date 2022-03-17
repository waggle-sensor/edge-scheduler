package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/spf13/cobra"
)

func init() {
	var (
		suspend bool
		force   bool
	)
	cmdRm := &cobra.Command{
		Use:              "rm [FLAGS] JOB_ID",
		Short:            "Remove or suspend a job",
		TraverseChildren: true,
		Args:             cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID := args[0]
			r := interfacing.NewHTTPRequest(serverHostString)
			q, err := url.ParseQuery("id=" + jobID + "&suspend=" + strconv.FormatBool(suspend) + "&force=" + strconv.FormatBool(force))
			if err != nil {
				return err
			}
			resp, err := r.RequestGet(fmt.Sprintf("api/v1/jobs/%s/rm", jobID), q)
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
	flags := cmdRm.Flags()
	flags.BoolVarP(&suspend, "suspend", "s", false, "Suspend the job")
	flags.BoolVarP(&force, "force", "f", false, "Remove or suspend the job forcefully")
	rootCmd.AddCommand(cmdRm)
}
