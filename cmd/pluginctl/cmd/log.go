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
		// printLogFunc, terminateLog, err := pluginCtl.PrintLog(name, followLog)
		// if err != nil {
		// 	return
		// } else {
		// 	c := make(chan os.Signal, 1)
		// 	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		// 	go printLogFunc()
		// 	for {
		// 		select {
		// 		case <-c:
		// 			logger.Debug.Println("Log terminated from user")
		// 			return
		// 		case <-terminateLog:
		// 			logger.Debug.Println("Log terminated from handler")
		// 			return
		// 		}
		// 	}
		// }

		// podLog, err := pluginctl.Log(name, followLog)
		// if err != nil {
		// 	logger.Error.Println("%s", err.Error())
		// } else {
		// 	c := make(chan os.Signal, 1)
		// 	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		// 	go func() {
		// 		buf := make([]byte, 2000)
		// 		for {
		// 			numBytes, err := podLog.Read(buf)
		// 			if numBytes == 0 {
		// 				break
		// 			}
		// 			if err == io.EOF {
		// 				break
		// 			} else if err != nil {
		// 				// return err
		// 			}
		// 			fmt.Println(string(buf[:numBytes]))
		// 		}
		// 		c <- nil
		// 	}()
		// 	<-c
		// 	podLog.Close()
		// }
	},
}
