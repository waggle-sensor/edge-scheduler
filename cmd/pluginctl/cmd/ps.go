package cmd

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/pluginctl"
	"github.com/spf13/cobra"
)

func init() {
	// cmdSub.Flags().StringVarP(&token, "token", "t", "", "Token to authenticate")
	// cmdSub.Flags().StringVarP(&job, "job", "j", "", "Description of job")
	rootCmd.AddCommand(cmdPs)
}

var cmdPs = &cobra.Command{
	Use:   "ps [APP_NAME]",
	Short: "Query plugin status",
	Run: func(cmd *cobra.Command, args []string) {
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			logger.Error.Println(err.Error())
		}
		list, err := pluginCtl.ResourceManager.ListPlugin()
		if err != nil {
			logger.Error.Println(err.Error())
		}
		for _, plugin := range list.Items {
			fmt.Printf("%s\n", plugin.Name)
		}
	},
}
