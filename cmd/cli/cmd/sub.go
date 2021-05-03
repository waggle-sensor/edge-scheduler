package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	cmdSub.Flags().StringVarP(&rules, "token", "t", "", "Token to authenticate")
	cmdSub.Flags().StringVarP(&rules, "rules", "r", "", "Path to Science Rules")
	rootCmd.AddCommand(cmdSub)
}

var (
	rules string
	token string
	nodeList 
)

var cmdSub = &cobra.Command{
	Use:   "submit [string to print]",
	Short: "Submit a job to SES",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Print: " + strings.Join(args, " "))
		fmt.Println("path " + rules)
		fmt.Println("token" + token)
	},
}
