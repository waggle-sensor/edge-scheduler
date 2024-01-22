package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
)

func init() {
	// cmdRm.Flags().BoolVarP(&followLog, "follow", "f", false, "Specified if logs should be streamed")
	rootCmd.AddCommand(cmdRm)
}

var cmdRm = &cobra.Command{
	Use:              "rm APP_NAME",
	Short:            "Remove plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		name := args[0]
		logger.Debug.Printf("args: %v", args)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return
		}
		err = pluginCtl.TerminatePlugin(name)
		if err != nil {
			return
		}
		fmt.Printf("Terminated the plugin %s successfully\n", name)
		return
	},
}
