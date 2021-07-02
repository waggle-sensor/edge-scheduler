package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	cmdSub.Flags().StringVarP(&token, "token", "t", "", "Token to authenticate")
	cmdSub.Flags().StringVarP(&job, "job", "j", "", "Description of job")
	// cmdSub.Flags().StringVarP(&rules, "rules", "r", "", "Path to Science Rules")
	// cmdSub.Flags().StringVarP(&nodeList, "node-list", "n", "", "Node list")
	rootCmd.AddCommand(cmdSub)
}

var (
	plugins    []string
	job        string
	rules      string
	token      string
	nodeList   string
	nodeFilter string
)

var cmdSub = &cobra.Command{
	Use:   "submit [string to print]",
	Short: "Submit a job to SES",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
		jobData, _ := ioutil.ReadFile(job)

		fmt.Println(string(jobData))

		resp, err := http.Post("http://localhost:9770/api/v1/submit", "applications/yaml", bytes.NewBuffer(jobData))
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(resp.Status))
			// var res map[string]interface{}

			// json.NewDecoder(resp.Body).Decode(&res)
			r, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(r))
		}

		// fmt.Println("path " + rules)
		// fmt.Println("token" + token)
		// fmt.Println("list" + nodeList)
	},
}
