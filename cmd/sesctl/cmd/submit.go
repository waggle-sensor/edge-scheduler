package cmd

import "github.com/spf13/cobra"

func init() {
	flags := cmdSubmit.Flags()
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	// flags.StringVarP(&job, "job", "j", "sage", "Specify job name")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	rootCmd.AddCommand(cmdSubmit)
}

var cmdSubmit = &cobra.Command{
	Use:              "run [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Run a plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
