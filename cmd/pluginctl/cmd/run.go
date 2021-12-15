package cmd

import (
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/spf13/cobra"
)

func init() {
	cmdRun.Flags().StringVarP(&name, "name", "n", "", "Specify plugin name")
	cmdRun.Flags().StringVar(&node, "node", "", "run plugin on node")
	cmdRun.Flags().StringVarP(&job, "job", "j", "sage", "Specify job name")
	cmdRun.Flags().StringVar(&selectorStr, "selector", "", "Specify where plugin can run")
	cmdRun.Flags().BoolVarP(&privileged, "privileged", "p", false, "Deploy as privileged plugin")
	rootCmd.AddCommand(cmdRun)
}

var cmdRun = &cobra.Command{
	Use:              "run [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Run a plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		logger.Debug.Printf("name: %s", name)
		logger.Debug.Printf("selector: %s", selectorStr)
		logger.Debug.Printf("args: %v", args)

	},
}
