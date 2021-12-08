package cmd

import (
	"fmt"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"github.com/spf13/cobra"
)

func init() {
	cmdRm.Flags().BoolVarP(&followLog, "follow", "f", false, "Specified if logs should be streamed")
	rootCmd.AddCommand(cmdRm)
}

var cmdRm = &cobra.Command{
	Use:              "rm APP_NAME",
	Short:            "Remove plugin",
	TraverseChildren: true,
	Args:             cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		name = args[0]
		logger.Debug.Printf("args: %v", args)

		resourceManager, err := nodescheduler.NewK3SResourceManager("", false, kubeconfig, nil, false)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
		}
		resourceManager.Namespace = "default"
		err = resourceManager.TerminatePlugin(name)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
		}
	},
}
