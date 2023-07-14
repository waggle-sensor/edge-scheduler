package pluginctl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler"
	"gopkg.in/yaml.v2"
	apiv1 "k8s.io/api/core/v1"
)

const pluginctlJob = "Pluginctl"

type MetricsServerConfig struct {
	InfluxDBTokenPath string
}

type MetricsServer struct {
	config *MetricsServerConfig
	Client influxdb2.Client
}

type PluginCtl struct {
	kubeConfig      string
	MetricsServer   *MetricsServer
	ResourceManager *nodescheduler.ResourceManager
	DryRun          bool
}

// Deployment holds the config pluginctl uses to deploy plugins
type Deployment struct {
	Name                   string
	SelectorString         string
	Node                   string
	Entrypoint             string
	Privileged             bool
	PluginImage            string
	PluginArgs             []string
	EnvVarString           []string
	EnvFromFile            string
	DevelopMode            bool
	Type                   string
	ResourceString         string
	EnablePluginController bool
	ForceToUpdate          bool
}

func NewPluginCtl(kubeconfig string) (*PluginCtl, error) {
	resourceManager, err := nodescheduler.NewK3SResourceManager(false, kubeconfig, "pluginctl", false)
	if err != nil {
		return nil, err
	}
	resourceManager.Namespace = "default"
	return &PluginCtl{
		kubeConfig:      kubeconfig,
		ResourceManager: resourceManager,
	}, nil
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

func (p *PluginCtl) GetMetrcisServerURL() (string, error) {
	ip, err := p.ResourceManager.GetServiceClusterIP("wes-node-influxdb", "default")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:8086", ip), nil
}

func (p *PluginCtl) ConnectToMetricsServer(config MetricsServerConfig) error {
	var path string
	if strings.HasPrefix(config.InfluxDBTokenPath, "~/") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, config.InfluxDBTokenPath[2:])
	} else {
		path = config.InfluxDBTokenPath
	}
	tokenBlob, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read token at %s: %s", config.InfluxDBTokenPath, err)
	}
	token := string(tokenBlob)
	token = strings.TrimSpace(token)
	if len(token) == 0 {
		return fmt.Errorf("Token is empty")
	}
	influxURL, err := p.GetMetrcisServerURL()
	if err != nil {
		return err
	}
	if p.MetricsServer == nil {
		p.MetricsServer = &MetricsServer{}
	}
	p.MetricsServer.Client = influxdb2.NewClientWithOptions(
		influxURL,
		string(token),
		influxdb2.DefaultOptions().SetHTTPRequestTimeout(60))
	// Ping to the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := p.MetricsServer.Client.Ping(ctx)
	if err != nil {
		return err
	}
	if !result {
		return fmt.Errorf("The server %s is not running", influxURL)
	}
	return nil
}

func (p *PluginCtl) Deploy(dep *Deployment) (string, error) {
	selector, err := ParseSelector(dep.SelectorString)
	if err != nil {
		return "", fmt.Errorf("Failed to parse selector %q: %s", dep.SelectorString, err.Error())
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
	resource, err := ParseSelector(dep.ResourceString)
	if err != nil {
		return "", fmt.Errorf("Failed to parse resource %q: %s", dep.ResourceString, err.Error())
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
	pluginRuntime := datatype.PluginRuntime{
		Plugin: datatype.Plugin{
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
				Resource:    resource,
			},
		},
		EnablePluginController: dep.EnablePluginController,
	}
	switch dep.Type {
	case "pod":
		pod, err := p.ResourceManager.CreatePodTemplate(&pluginRuntime)
		if err != nil {
			return "", err
		}
		if p.DryRun {
			return pod.Name, writeResourceYAML(os.Stdout, "core/v1", "Pod", pod)
		} else {
			return pod.Name, p.ResourceManager.UpdatePod(pod, dep.ForceToUpdate)
		}
	case "job":
		job, err := p.ResourceManager.CreateJobTemplate(&pluginRuntime)
		if err != nil {
			return "", err
		}
		if p.DryRun {
			return job.Name, writeResourceYAML(os.Stdout, "batch/v1", "Job", job)
		} else {
			return job.Name, p.ResourceManager.UpdateJob(job, dep.ForceToUpdate)
		}
	case "deployment":
		deployment, err := p.ResourceManager.CreateDeploymentTemplate(&pluginRuntime)
		if err != nil {
			return "", err
		}
		if p.DryRun {
			return deployment.Name, writeResourceYAML(os.Stdout, "apps/v1", "Deployment", deployment)
		} else {
			return deployment.Name, p.ResourceManager.UpdateDeployment(deployment, dep.ForceToUpdate)
		}
	case "daemonset":
		daemonSet, err := p.ResourceManager.CreateDaemonSetTemplate(&pluginRuntime)
		if err != nil {
			return "", err
		}
		if p.DryRun {
			return daemonSet.Name, writeResourceYAML(os.Stdout, "apps/v1", "DaemonSet", daemonSet)
		} else {
			return daemonSet.Name, p.ResourceManager.UpdateDaemonSet(daemonSet, dep.ForceToUpdate)
		}
	default:
		return "", fmt.Errorf("Unknown type %q for plugin", dep.Type)
	}
}

func writeResourceYAML(w io.Writer, apiVersion string, kind string, resource interface{}) error {
	cleaned, err := cleanResource(resource)
	if err != nil {
		return err
	}

	cleaned["apiVersion"] = apiVersion
	cleaned["kind"] = kind
	delete(cleaned, "status")

	return yaml.NewEncoder(w).Encode(cleaned)
}

// cleanResource is a helper function which leverages the fact that k8s types use json
// tags to omit empty fields. unfortunately, using yaml directly leaves all these in.
func cleanResource(resource interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}

	var cleaned map[string]interface{}

	if err := json.Unmarshal(b, &cleaned); err != nil {
		return nil, err
	}

	return cleaned, nil
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
	podLog, err := p.ResourceManager.GetPodLogHandler(pluginName, follow)
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
	return p.ResourceManager.TerminatePod(pluginName)
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
			if err != nil {
				status = "ERROR"
			} else {
				status = string(podPhase)
				duration = time.Since(plugin.Status.StartTime.Time).String()
			}
		}
		formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, plugin.Name, maxLengthStatus+3, status, maxLengthStartTime+3, startTime, maxLengthDuration+3, duration)
	}
	return
}

func (p *PluginCtl) GetPerformanceData(s time.Time, e time.Time, pluginName string) error {
	logger.Info.Println("Start gathering performance data...")
	var q string
	if pluginName == "" {
		logger.Info.Println("no plugin name is given. all plugins are subject for search. the output name will be all.csv")
		q = fmt.Sprintf(`
		from(bucket:"waggle")
		  |> range(start: %s, stop: %s)
		  |> filter(fn: (r) => r._measurement =~ /sys.plugin.perf.*/)`,
			s.Format(time.RFC3339),
			e.Format(time.RFC3339))
		pluginName = "all"
	} else {
		q = fmt.Sprintf(`
		from(bucket:"waggle")
		  |> range(start: %s, stop: %s)
		  |> filter(fn: (r) => r["task"] =~ /^%s*/)
		  |> filter(fn: (r) => r._measurement =~ /sys.plugin.perf.*/)`,
			s.Format(time.RFC3339),
			e.Format(time.RFC3339),
			pluginName)
	}
	logger.Debug.Println(q)
	// TODO: We will need an error handling on whether the metrics server with config is set
	queryAPI := p.MetricsServer.Client.QueryAPI("waggle")
	f, err := os.OpenFile(fmt.Sprintf("%s.csv", pluginName), os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	result, err := queryAPI.QueryRaw(context.Background(), q, &domain.Dialect{})
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
	return nil
}
