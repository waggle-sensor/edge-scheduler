package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/pluginctl"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func init() {
	flags := cmdRun.Flags()
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	// flags.StringVarP(&job, "job", "j", "sage", "Specify job name")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
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
		defer pluginCtl.TerminatePlugin(pluginName)

		fmt.Printf("Launched the plugin %s successfully \n", pluginName)
		for {
			pluginStatus, err := pluginCtl.GetPluginStatus(pluginName)
			if err != nil {
				return err
			}
			if pluginStatus != apiv1.PodPending {
				break
			}
			logger.Info.Printf("Plugin is in %q state. Waiting...", pluginStatus)
			time.Sleep(2 * time.Second)
		}
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			watcher, err := pluginCtl.ResourceManager.WatchJob(pluginName, pluginCtl.ResourceManager.Namespace, 0)
			if err != nil {
				logger.Error.Printf("%q", err.Error())
				c <- nil
			}
			chanGoal := watcher.ResultChan()
			for {
				event := <-chanGoal
				switch event.Type {
				case watch.Added, watch.Deleted, watch.Modified:
					switch obj := event.Object.(type) {
					case *batchv1.Job:
						if len(obj.Status.Conditions) > 0 {
							logger.Debug.Printf("%s: %s", event.Type, obj.Status.Conditions[0].Type)
							switch obj.Status.Conditions[0].Type {
							case batchv1.JobComplete, batchv1.JobFailed:
								c <- nil
							}
						} else {
							logger.Debug.Printf("job unexpectedly missing status conditions: %v", obj)
						}
					default:
						logger.Debug.Printf("%s: %s", event.Type, "UNKNOWN")
					}
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
