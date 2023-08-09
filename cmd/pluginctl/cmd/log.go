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
	cmdLog.Flags().BoolVarP(&followLog, "follow", "f", false, "Specified if logs should be streamed")
	rootCmd.AddCommand(cmdLog)
}

var cmdLog = &cobra.Command{
	Use:              "logs [FLAGS] PLUGIN_NAME",
	Short:            "Print logs of a plugin",
	TraverseChildren: true,
	// DisableFlagParsing: true,
	Args: cobra.MinimumNArgs(1),
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
		return k.Execute()

		// c := make(chan os.Signal, 1)
		// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		// go func() {
		// 	watcher, err := pluginCtl.ResourceManager.WatchPod(name, pluginCtl.ResourceManager.Namespace, 0)
		// 	if err != nil {
		// 		logger.Error.Printf("%q", err.Error())
		// 		c <- nil
		// 	}
		// 	chanEvent := watcher.ResultChan()
		// 	for event := range chanEvent {
		// 		switch event.Type {
		// 		case watch.Modified:
		// 			_pod := event.Object.(*v1.Pod)
		// 			switch _pod.Status.Phase {
		// 			case v1.PodSucceeded, v1.PodFailed:
		// 				logger.Debug.Printf("%s: %s", event.Type, _pod.Status.Phase)
		// 				c <- nil
		// 			}
		// 		case watch.Deleted:
		// 			logger.Error.Printf("Plugin deleted unexpectedly")
		// 			c <- nil
		// 		}
		// 	}
		// }()
		// printLogFunc, terminateLog, err := pluginCtl.PrintLog(name, false)
		// if err != nil {
		// 	return err
		// } else {
		// 	go printLogFunc()
		// 	for {
		// 		select {
		// 		case <-c:
		// 			logger.Debug.Println("Log terminated from user side")
		// 			return nil
		// 		case <-terminateLog:
		// 			logger.Debug.Println("Log terminated from handler")
		// 			return nil
		// 		}
		// 	}
		// }
	},
}
