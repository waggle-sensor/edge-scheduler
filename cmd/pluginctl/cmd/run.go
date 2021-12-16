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
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		logger.Debug.Printf("name: %s", name)
		logger.Debug.Printf("selector: %s", selectorStr)
		logger.Debug.Printf("args: %v", args)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return err
		}
		pluginName, err := pluginCtl.Deploy(name, selectorStr, node, privileged, args[0], args[1:])
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
			watcher, err := pluginCtl.ResourceManager.WatchJob(pluginName, pluginCtl.ResourceManager.Namespace)
			if err != nil {
				logger.Error.Printf("%q", err.Error())
				c <- nil
			}
			chanGoal := watcher.ResultChan()
			for {
				event := <-chanGoal
				switch event.Type {
				case watch.Added:
					fallthrough
				case watch.Deleted:
					fallthrough
				case watch.Modified:
					job := event.Object.(*batchv1.Job)
					if len(job.Status.Conditions) > 0 {
						logger.Debug.Printf("%s: %s", event.Type, job.Status.Conditions[0].Type)
						switch job.Status.Conditions[0].Type {
						case batchv1.JobComplete:
							fallthrough
						case batchv1.JobFailed:
							c <- nil
						}
					} else {
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
