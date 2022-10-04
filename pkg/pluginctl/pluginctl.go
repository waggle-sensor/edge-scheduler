package pluginctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler"
	apiv1 "k8s.io/api/core/v1"
)

const pluginctlJob = "Pluginctl"

type PluginCtl struct {
	ResourceManager *nodescheduler.ResourceManager
}

// Deployment holds the config pluginctl uses to deploy plugins
type Deployment struct {
	Name           string
	SelectorString string
	Node           string
	Entrypoint     string
	Privileged     bool
	PluginImage    string
	PluginArgs     []string
	EnvVarString   []string
	EnvFromFile    string
	DevelopMode    bool
}

func NewPluginCtl(kubeconfig string) (*PluginCtl, error) {
	resourceManager, err := nodescheduler.NewK3SResourceManager(false, kubeconfig, "pluginctl", false)
	if err != nil {
		return nil, err
	}
	resourceManager.Namespace = "default"
	return &PluginCtl{ResourceManager: resourceManager}, nil
}

func parseEnv(envs []string) (map[string]string, error) {
	items := map[string]string{}
	for _, s := range envs {
		s = strings.TrimSpace(s)
		if s == "" {
			return items, nil
		}
		k, v, err := parseSelectorTerm(s)
		if err != nil {
			return items, err
		}
		items[k] = v
	}
	return items, nil
}

func (p *PluginCtl) Deploy(dep *Deployment) (string, error) {
	selector, err := ParseSelector(dep.SelectorString)
	if err != nil {
		return "", fmt.Errorf("Failed to parse selector %q", err.Error())
	}
	if dep.EnvFromFile != "" {
		logger.Debug.Printf("Reading env file %q...", dep.EnvFromFile)
		file, err := os.Open(dep.EnvFromFile)
		if err != nil {
			return "", fmt.Errorf("Failed to open env-from file %q: %s", dep.EnvFromFile, err.Error())
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			dep.EnvVarString = append(dep.EnvVarString, scanner.Text())
		}
	}
	envs, err := parseEnv(dep.EnvVarString)
	if err != nil {
		return "", fmt.Errorf("Failed to parse env %q", err.Error())
	}
	plugin := datatype.Plugin{
		Name: dep.Name,
		PluginSpec: &datatype.PluginSpec{
			Privileged:  dep.Privileged,
			Node:        dep.Node,
			Image:       dep.PluginImage,
			Args:        dep.PluginArgs,
			Job:         pluginctlJob,
			Selector:    selector,
			Entrypoint:  dep.Entrypoint,
			Env:         envs,
			DevelopMode: dep.DevelopMode,
		},
	}
	job, err := p.ResourceManager.CreateJob(&plugin)
	if err != nil {
		return "", err
	}
	deployedJob, err := p.ResourceManager.RunPlugin(job)
	return deployedJob.Name, err
}

// RunAsync runs given plugin and reports plugin status changes back to caller
// The function is expected to be called asynchronously, i.e. go RunAsync()
func (p *PluginCtl) RunAsync(dep *Deployment, chEvent chan<- datatype.Event, out *os.File) {
	// Run plugin
	pluginName, err := p.Deploy(dep)
	if err != nil {
		eventBuilder := datatype.NewEventBuilder(datatype.EventFailure).AddReason(err.Error())
		chEvent <- eventBuilder.Build()
		return
	}
	defer p.TerminatePlugin(pluginName)
	eventBuilder := datatype.NewEventBuilder(datatype.EventPluginStatusLaunched).AddEntry("plugin_name", pluginName)
	chEvent <- eventBuilder.Build()
	pluginStartT := time.Now().UTC()
	// TODO: We will need to capture when the user does Ctrl + C to kill the caller
	// Check if the pod is running
	for {
		pluginStatus, err := p.GetPluginStatus(pluginName)
		if err != nil {
			fmt.Fprintln(out, "Waiting for plugin to run...")
		} else {
			if pluginStatus != apiv1.PodPending {
				break
			}
			fmt.Fprintf(out, "Plugin is in %q state. Waiting...\n", pluginStatus)
		}
		time.Sleep(2 * time.Second)
	}
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// go func() {
	// 	watcher, err := p.ResourceManager.WatchJob(pluginName, p.ResourceManager.Namespace, 0)
	// 	if err != nil {
	// 		logger.Error.Printf("%q", err.Error())
	// 		c <- nil
	// 	}
	// 	chanGoal := watcher.ResultChan()
	// 	for {
	// 		event := <-chanGoal
	// 		switch event.Type {
	// 		case watch.Added, watch.Deleted, watch.Modified:
	// 			switch obj := event.Object.(type) {
	// 			case *batchv1.Job:
	// 				if len(obj.Status.Conditions) > 0 {
	// 					logger.Debug.Printf("%s: %s", event.Type, obj.Status.Conditions[0].Type)
	// 					switch obj.Status.Conditions[0].Type {
	// 					case batchv1.JobComplete, batchv1.JobFailed:
	// 						c <- nil
	// 					}
	// 				} else {
	// 					logger.Debug.Printf("job unexpectedly missing status conditions: %v", obj)
	// 				}
	// 			default:
	// 				logger.Debug.Printf("%s: %s", event.Type, "UNKNOWN")
	// 			}
	// 		}
	// 	}
	// }()

	printLogFunc, terminateLog, err := p.PrintLog(pluginName, true)
	if err != nil {
		eventBuilder := datatype.NewEventBuilder(datatype.EventFailure).AddReason(err.Error())
		chEvent <- eventBuilder.Build()
		return
	}
	go printLogFunc()
	for {
		select {
		// case <-c:
		// 	logger.Debug.Println("Log terminated from user side")
		// 	pluginEndT := time.Now().UTC()
		// 	logger.Info.Println(pluginStartT)
		// 	logger.Info.Println(pluginEndT)
		// 	return nil
		case <-terminateLog:
			logger.Debug.Println("Log terminated from handler")
			pluginEndT := time.Now().UTC()
			logger.Info.Println(pluginStartT)
			logger.Info.Println(pluginEndT)
		}
	}
}

func (p *PluginCtl) PrintLog(pluginName string, follow bool) (func(), chan os.Signal, error) {
	podLog, err := p.ResourceManager.GetPluginLog(pluginName, follow)
	if err != nil {
		return nil, nil, err
	}
	flag := make(chan os.Signal, 1)
	return func() {
		buf := make([]byte, 2000)
		for {
			select {
			case <-flag:
				logger.Debug.Printf("Log handler closed by func")
				podLog.Close()
				return
			default:
				numBytes, err := podLog.Read(buf)
				if numBytes == 0 {
					continue
				}
				if err != nil {
					if err == io.EOF {
						fmt.Printf(string(buf[:numBytes]))
					}
					logger.Debug.Printf("Log handler closed by error: %s", err.Error())
					podLog.Close()
					flag <- nil
					return
				}
				fmt.Printf(string(buf[:numBytes]))
			}
		}
	}, flag, nil
}

func (p *PluginCtl) TerminatePlugin(pluginName string) error {
	return p.ResourceManager.TerminateJob(pluginName)
}

func (p *PluginCtl) GetPluginStatus(name string) (apiv1.PodPhase, error) {
	return p.ResourceManager.GetPluginStatus(name)
}

// GetPlugins returns list of plugins. It is assumed that each Kubernetes job handling
// a plugin has only one Kubernetes pod, so only the first pod associated to the job is
// considered when printing state of the job.
func (p *PluginCtl) GetPlugins() (formattedList string, err error) {
	list, err := p.ResourceManager.ListJobs()
	if err != nil {
		return
	}
	var (
		maxLengthName      int = 0
		maxLengthStatus    int = len("succeeded")
		maxLengthStartTime int = 23
		maxLengthDuration  int = 4
	)
	for _, plugin := range list.Items {
		if strings.Contains(plugin.Name, "wes") {
			continue
		}
		if len(plugin.Name) > maxLengthName {
			maxLengthName = len(plugin.Name)
		}
	}
	formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, "NAME", maxLengthStatus+3, "STATUS", maxLengthStartTime+3, "START_TIME", maxLengthDuration+3, "RUNNING_TIME")
	var (
		startTime string
		duration  string
		status    string
	)
	for _, plugin := range list.Items {
		if strings.Contains(plugin.Name, "wes") {
			continue
		}
		startTime = ""
		duration = ""
		status = "UNKNOWN"
		if plugin.Status.StartTime != nil {
			// NOTE: Time format in Go is so special. https://pkg.go.dev/time#Time.Format
			startTime = plugin.Status.StartTime.Time.UTC().Format("2006/01/02 15:04:05 MST")
		}
		if len(plugin.Status.Conditions) > 0 {
			status = string(plugin.Status.Conditions[0].Type)
			if plugin.Status.CompletionTime != nil {
				duration = plugin.Status.CompletionTime.Sub(plugin.Status.StartTime.Time).String()
			}
		} else {
			podPhase, err := p.ResourceManager.GetPluginStatus(plugin.Name)
			if err == nil {
				status = string(podPhase)
			}
			duration = time.Since(plugin.Status.StartTime.Time).String()
		}
		formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, plugin.Name, maxLengthStatus+3, status, maxLengthStartTime+3, startTime, maxLengthDuration+3, duration)
	}
	return
}
