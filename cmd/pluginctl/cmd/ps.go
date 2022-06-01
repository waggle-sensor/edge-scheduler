package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
)

func init() {
	// cmdSub.Flags().StringVarP(&token, "token", "t", "", "Token to authenticate")
	// cmdSub.Flags().StringVarP(&job, "job", "j", "", "Description of job")
	rootCmd.AddCommand(cmdPs)
}

var cmdPs = &cobra.Command{
	Use:   "ps [APP_NAME]",
	Short: "Query plugin list",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return
		}
		list, err := pluginCtl.GetPlugins()
		if err != nil {
			return
		}
		fmt.Println(list)
		return
	},
}
