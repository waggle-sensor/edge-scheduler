package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	// cmdSub.Flags().StringVarP(&token, "token", "t", "", "Token to authenticate")
	// cmdSub.Flags().StringVarP(&job, "job", "j", "", "Description of job")
	rootCmd.AddCommand(cmdPs)
}

var cmdPs = &cobra.Command{
	Use:   "ps job_id",
	Short: "Query job status",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
		url := fmt.Sprintf("http://localhost:9770/api/v1/jobs/%s/status", args[0])
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(resp.Status))
			// var res map[string]interface{}

			// json.NewDecoder(resp.Body).Decode(&res)
			r, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(r))
		}
	},
}
