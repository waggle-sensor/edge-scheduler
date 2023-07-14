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

var metricsServerConfig pluginctl.MetricsServerConfig

var (
	start string
	end   string
)

func init() {
	flags := cmdProfileRun.Flags()
	flags.StringVarP(&deployment.Name, "name", "n", "", "Specify plugin name")
	flags.StringVar(&deployment.Node, "node", "", "run plugin on node")
	flags.StringVar(&deployment.SelectorString, "selector", "", "Specify where plugin can run")
	flags.StringVar(&deployment.Entrypoint, "entrypoint", "", "Specify command to run inside plugin")
	flags.BoolVarP(&deployment.Privileged, "privileged", "p", false, "Deploy as privileged plugin")
	flags.StringSliceVarP(&deployment.EnvVarString, "env", "e", []string{}, "Set environment variables")
	flags.StringVarP(&deployment.EnvFromFile, "env-from", "", "", "Set environment variables from file")
	flags.BoolVar(&deployment.DevelopMode, "develop", false, "Enable the following development time features: access to wan network")
	flags.StringVar(&deployment.ResourceString, "resource", "", "Specify resource requirement for running the plugin")
	flags.StringVar(&metricsServerConfig.InfluxDBTokenPath, "influxdb-token-path", getenv("INFLUXDB_TOKEN_PATH", "~/.influxdb2/token"), "Path to valid token to access InfluxDB")
	cmdProfile.AddCommand(cmdProfileRun)
	flags = cmdProfileGet.Flags()
	flags.StringVar(&start, "start", "", "Search data since the start time in UTC. Should be formatted as RFC3339")
	flags.StringVar(&end, "end", "", "Search data until the end time in UTC. Should be formatted as RFC3339")
	cmdProfileGet.MarkFlagRequired("start")
	cmdProfileGet.MarkFlagRequired("end")
	cmdProfile.AddCommand(cmdProfileGet)
	rootCmd.AddCommand(cmdProfile)
}

var cmdProfileRun = &cobra.Command{
	Use:              "run [FLAGS] PLUGIN_IMAGE [-- PLUGIN ARGUMENTS]",
	Short:            "run and profile performance of a plugin",
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
		// Metrics server will provide performance data after the run
		err = pluginCtl.ConnectToMetricsServer(metricsServerConfig)
		if err != nil {
			logger.Error.Printf("Failed to check metrics server: %s", err.Error())
			logger.Error.Println("Abort profiling due to the error")
			return err
		}
		// in profiling we always enable plugin controller to collect performance metrics
		deployment.EnablePluginController = true
		deployment.Type = "pod"
		pluginName, err := pluginCtl.Deploy(deployment)
		if err != nil {
			return err
		}
		defer pluginCtl.TerminatePlugin(pluginName)
		startT := time.Now().UTC()
		fmt.Printf("Launched the plugin %s successfully \n", pluginName)
		maxErrorCount := 5
		errorCount := 0
		for {
			pluginStatus, err := pluginCtl.GetPluginStatus(pluginName)
			if err != nil {
				errorCount += 1
				logger.Debug.Printf("Failed to get plugin status: %s", err.Error())
				if errorCount > maxErrorCount {
					return fmt.Errorf("Failed to get plugin status %s", err.Error())
				}
				logger.Debug.Printf("Retrying with attempt count %d", errorCount)
			} else {
				if pluginStatus == apiv1.PodRunning {
					break
				}
				logger.Info.Printf("Plugin is in %q state. Waiting...", pluginStatus)
			}
			time.Sleep(2 * time.Second)
		}
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
					endT := time.Now().UTC()
					logger.Info.Printf("Plugin took %s to finish", endT.Sub(startT).String())
					pluginCtl.GetPerformanceData(startT, endT, pluginName)
					return nil
				case <-terminateLog:
					logger.Debug.Println("Log terminated from handler")
					endT := time.Now().UTC()
					logger.Info.Printf("Plugin took %s to finish", endT.Sub(startT).String())
					pluginCtl.GetPerformanceData(startT, endT, pluginName)
					return nil
				}
			}
		}
	},
}

var cmdProfileGet = &cobra.Command{
	Use:              "get [FLAGS] PLUGIN_K3S_POD_NAME",
	Short:            "get existing performance data of a plugin",
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pluginName string
		if len(args) == 0 {
			pluginName = ""
		} else {
			pluginName = args[0]
		}
		logger.Debug.Printf("kubeconfig: %s", kubeconfig)
		pluginCtl, err := pluginctl.NewPluginCtl(kubeconfig)
		if err != nil {
			return err
		}
		// Metrics server will provide performance data after the run
		err = pluginCtl.ConnectToMetricsServer(metricsServerConfig)
		if err != nil {
			logger.Error.Printf("Failed to check metrics server: %s", err.Error())
			logger.Error.Println("Abort profiling due to the error")
			return err
		}
		s, err := time.Parse(time.RFC3339Nano, start)
		if err != nil {
			return err
		}
		e, err := time.Parse(time.RFC3339Nano, end)
		if err != nil {
			return err
		}
		pluginCtl.GetPerformanceData(s, e, pluginName)
		return nil
	},
}

var cmdProfile = &cobra.Command{
	Use:   "profile",
	Short: "Profile performance of a plugin",
}
