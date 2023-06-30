package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func init() {
	flags := cmdRun.Flags()
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.StringSliceVarP(&deployment.EnvVarString, "env", "e", []string{}, "Set environment variables")
	flags.StringVarP(&deployment.EnvFromFile, "env-from", "", "", "Set environment variables from file")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	flags.StringVar(&deployment.ResourceString, "resource", "", "Specify resource requirement for running the plugin")
	rootCmd.AddCommand(cmdRun)
}

var cmdRun = &cobra.Command{
	Use:              "run [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Run a plugin",
	TraverseChildren: true,
	Args:             cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		deployment.PluginImage = args[0]
		deployment.PluginArgs = args[1:]
		// run always uses pod type
		deployment.Type = "pod"

		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		logger.Debug.Printf("deployment: %#v", deployment)

		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return err
		}

		pluginName, err := pluginCtl.Deploy(deployment)
		if err != nil {
			return err
		}
		fmt.Printf("Scheduled the plugin %s successfully \n", pluginName)
		maxErrorCount := 5
		errorCount := 0
	checkStatusLoop:
		for {
			pluginStatus, err := pluginCtl.GetPluginStatus(pluginName)
			if err != nil {
				errorCount += 1
				logger.Debug.Printf("Failed to get plugin status: %s", err.Error())
				if errorCount > maxErrorCount {
					return fmt.Errorf("Failed to get plugin status %s", err.Error())
				}
				logger.Debug.Printf("Retrying with attempt count %d", errorCount)
			}
			logger.Debug.Printf("Current pod status: %s", pluginStatus)
			switch pluginStatus {
			case apiv1.PodRunning, apiv1.PodSucceeded:
				break checkStatusLoop
			case apiv1.PodFailed:
				// TODO: We will need to extract error on why it was pending or failed.
				return fmt.Errorf("Plugin failed to run. The plugin remains in the system. Please investigate the problem and clean it up.")
			}
			logger.Info.Printf("Plugin is in %q state. Waiting...", pluginStatus)
			time.Sleep(2 * time.Second)
		}
		defer pluginCtl.TerminatePlugin(pluginName)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			watcher, err := pluginCtl.ResourceManager.WatchPod(pluginName, pluginCtl.ResourceManager.Namespace, 0)
			if err != nil {
				logger.Error.Printf("%q", err.Error())
				c <- nil
			}
			chanEvent := watcher.ResultChan()
			for event := range chanEvent {
				switch event.Type {
				case watch.Modified:
					_pod := event.Object.(*v1.Pod)
					switch _pod.Status.Phase {
					case v1.PodSucceeded, v1.PodFailed:
						logger.Debug.Printf("%s: %s", event.Type, _pod.Status.Phase)
						c <- nil
					}
				case watch.Deleted:
					logger.Error.Printf("Plugin deleted unexpectedly")
					c <- nil
				}
			}
		}()
		printLogFunc, terminateLog, err := pluginCtl.PrintLog(pluginName, true)
		if err != nil {
			return err
		} else {
			go printLogFunc()
			for {
				select {
				case <-c:
					logger.Debug.Println("Log terminated from user side")
					return nil
				case <-terminateLog:
					logger.Debug.Println("Log terminated from handler")
					return nil
				}
			}
		}
	},
}
