package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
)

var dryRun bool

func init() {
	flags := cmdDeploy.Flags()
	flags.BoolVar(&dryRun, "dry-run", false, "Output Kubernetes resource which would have been created.")
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.StringSliceVarP(&deployment.EnvVarString, "env", "e", []string{}, "Set environment variables")
	flags.StringVarP(&deployment.EnvFromFile, "env-from", "", "", "Set environment variables from file")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	flags.StringVar(&deployment.Type, "type", "job", "Type of the plugin. It is one of ['job', 'deployment', 'daemonset]. Default is 'job'.")
	flags.StringVar(&deployment.ResourceString, "resource", "", "Specify resource requirement for running the plugin")
	rootCmd.AddCommand(cmdDeploy)
}

var cmdDeploy = &cobra.Command{
	Use:              "deploy [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Deploy a plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deployment.PluginImage = args[0]
		deployment.PluginArgs = args[1:]

		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		logger.Debug.Printf("deployment: %#v", deployment)

		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return err
		}

		pluginCtl.DryRun = dryRun

		pluginName, err := pluginCtl.Deploy(deployment)
		if err != nil {
			return err
		}

		if !pluginCtl.DryRun {
			fmt.Printf("Launched the plugin %s successfully \n", pluginName)
			fmt.Printf("You may check the log: pluginctl logs %s\n", pluginName)
			fmt.Printf("To terminate the job: pluginctl rm %s\n", pluginName)
		}

		return nil
	},
}
