package cmd

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/pluginctl"
	"github.com/spf13/cobra"
)

func init() {
	cmdDeploy.Flags().StringVarP(&name, "name", "n", "", "Specify plugin name")
	cmdDeploy.Flags().StringVar(&node, "node", "", "run plugin on node")
	cmdDeploy.Flags().StringVar(&selectorStr, "selector", "", "Specify where plugin can run")
	cmdDeploy.Flags().BoolVarP(&privileged, "privileged", "p", false, "Deploy as privileged plugin")
	rootCmd.AddCommand(cmdDeploy)
}

var cmdDeploy = &cobra.Command{
	Use:              "deploy [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Deploy a plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		logger.Debug.Printf("name: %s", name)
		logger.Debug.Printf("selector: %s", selectorStr)
		logger.Debug.Printf("args: %v", args)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			logger.Error.Println(err.Error())
			return err
		}
		if pluginName, err := pluginCtl.Deploy(name, selectorStr, node, privileged, args[0], args[1:]); err != nil {
			logger.Error.Println(err.Error())
			return err
		} else {
			fmt.Printf("Launched the plugin %s successfully \n", pluginName)
			fmt.Printf("You may check the log: pluginctl log %s\n", pluginName)
			fmt.Printf("To terminate the job: pluginctl rm %s\n", pluginName)
		}
		return nil
	},
}
