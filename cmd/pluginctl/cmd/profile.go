package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/pluginctl"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func init() {
	flags := cmdProfile.Flags()
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.StringSliceVarP(&deployment.EnvVarString, "env", "e", []string{}, "Set environment variables")
	flags.StringVarP(&deployment.EnvFromFile, "env-from", "", "", "Set environment variables from file")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	rootCmd.AddCommand(cmdProfile)
}

func getPerformanceData(s time.Time, e time.Time, pluginName string) {
	logger.Info.Println("Start gathering performance data...")
	client := influxdb2.NewClient("http://localhost:8086", "o6o9QbxX85JBZEHzeob2t7y5ej5In9bqgG-skg967P-_XNHKCxSZsJU0xZZFUzjtXu0WLraJThNhqvq2JBFc2g==")
	queryAPI := client.QueryAPI("waggle")
	q := fmt.Sprintf(`
from(bucket:"waggle")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r._measurement == "prometheus_remote_write")
  |> filter(fn: (r) => r["pod"] =~ /^%s*/)`,
		s.Format(time.RFC3339),
		e.Format(time.RFC3339),
		pluginName)
	logger.Debug.Println(q)
	f, err := os.OpenFile(fmt.Sprintf("%s.csv", pluginName), os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	result, err := queryAPI.QueryRaw(context.Background(), q, influxdb2.DefaultDialect())
	if err == nil {
		n, err := f.WriteString(result)
		if err != nil {
			logger.Error.Printf("Failed to write performance data to a file: %s", err.Error())
		} else {
			logger.Debug.Printf("%d bytes written", n)
		}
	} else {
		logger.Error.Printf("Failed to obtain performance data: %s", err.Error())
	}

	q = fmt.Sprintf(`
from(bucket:"waggle")
  |> range(start: %s, stop: %s)
  |> filter(fn: (r) => r._measurement == "prometheus_remote_write")
  |> filter(fn: (r) => r._field == "gpu_average_load1s")`,
		s.Format(time.RFC3339),
		e.Format(time.RFC3339))
	logger.Debug.Println(q)
	result, err = queryAPI.QueryRaw(context.Background(), q, influxdb2.DefaultDialect())
	if err == nil {
		n, err := f.WriteString(result)
		if err != nil {
			logger.Error.Printf("Failed to write performance data to a file: %s", err.Error())
		} else {
			logger.Debug.Printf("%d bytes written", n)
		}
	} else {
		logger.Error.Printf("Failed to obtain performance data: %s", err.Error())
	}
}

var cmdProfile = &cobra.Command{
	Use:              "profile [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "Profile performance of a plugin",
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
		startT := time.Now().UTC()
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
					endT := time.Now().UTC()
					logger.Info.Printf("Plugin took %s to finish", endT.Sub(startT).String())
					getPerformanceData(startT, endT, pluginName)
					return nil
				case <-terminateLog:
					logger.Debug.Println("Log terminated from handler")
					endT := time.Now().UTC()
					logger.Info.Printf("Plugin took %s to finish", endT.Sub(startT).String())
					getPerformanceData(startT, endT, pluginName)
					return nil
				}
			}
		}

	},
}
