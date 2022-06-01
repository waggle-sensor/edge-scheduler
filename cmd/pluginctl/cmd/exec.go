package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubectl "k8s.io/kubectl/pkg/cmd"
)

func init() {
	cmdExec.Flags().BoolVarP(&stdin, "stdin", "i", false, "Pass stdin to the container")
	cmdExec.Flags().BoolVarP(&tty, "tty", "t", false, "Stdin is a TTY")
	rootCmd.AddCommand(cmdExec)
}

var cmdExec = &cobra.Command{
	Use:                   "exec [FLAGS] PLUGIN_NAME [-- COMMAND ARGUMENTS]",
	Short:                 "Execute a command on plugin",
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return
		}
		name, err := pluginCtl.ResourceManager.GetPodName(args[0])
		if err != nil {
			return
		}
		// name = args[0]
		// logger.Debug.Printf("args: %v", args)
		c := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
		c.KubeConfig = &kubeconfig
		for i := 0; i < len(os.Args); i++ {
			if os.Args[i] == args[0] {
				os.Args[i] = name
				break
			}
		}
		k := kubectl.NewDefaultKubectlCommandWithArgs(kubectl.KubectlOptions{
			PluginHandler: kubectl.NewDefaultPluginHandler([]string{"kubectl"}),
			Arguments:     os.Args,
			ConfigFlags:   c,
			IOStreams:     genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		})
		// k.SetArgs([]string{"exec", "-n", "default", "-it", "node-influxdb-77bb74f689-nr5zm", "--", "/bin/date"})
		return k.Execute()
		// pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		// if err != nil {
		// 	return
		// }
		// err = pluginCtl.TerminatePlugin(name)
		// if err != nil {
		// 	return
		// }
		// fmt.Printf("Terminated the plugin %s successfully\n", name)
		// return
	},
}
