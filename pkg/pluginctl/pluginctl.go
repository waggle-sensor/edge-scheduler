package pluginctl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	v1 "k8s.io/api/batch/v1"
)

type PluginCtl struct {
	ResourceManager *nodescheduler.ResourceManager
}

func NewPluginCtl(kubeconfig string) (*PluginCtl, error) {
	resourceManager, err := nodescheduler.NewK3SResourceManager("", false, kubeconfig, nil, false)
	if err != nil {
		return nil, err
	}
	resourceManager.Namespace = "default"
	return &PluginCtl{ResourceManager: resourceManager}, nil
}

var validNamePattern = regexp.MustCompile("^[a-z0-9-]+$")

func jobNameForSpec(spec *datatype.PluginSpec) (string, error) {
	// if no given name for the plugin, use PLUGIN-VERSION-INSTANCE format for name
	// INSTANCE is calculated as Sha256("DOMAIN/PLUGIN:VERSION&ARGUMENTS") and
	// take the first 8 hex letters.
	// NOTE: if multiple plugins with the same version and arguments are given for
	//       the same domain, only one deployment will be applied to the cluster
	// NOTE2: To comply with RFC 1123 for Kubernetes object name, only lower alphanumeric
	//        characters with '-' is allowed
	if spec.Name != "" {
		jobName := strings.Join([]string{spec.Name, strconv.FormatInt(time.Now().Unix(), 10)}, "-")
		if !validNamePattern.MatchString(jobName) {
			return "", fmt.Errorf("plugin name must consist of alphanumeric characters with '-' RFC1123")
		}
		return jobName, nil
	}
	return generateJobNameForSpec(spec)
}

// generateJobNameForSpec generates a consistent name for a Spec.
//
// Very important note from: https://pkg.go.dev/encoding/json#Marshal
//
// Map values encode as JSON objects. The map's key type must either be a string, an integer type,
// or implement encoding.TextMarshaler. The map keys are sorted and used as JSON object keys by applying
// the following rules, subject to the UTF-8 coercion described for string values above:
//
// The "map keys are sorted" bit is important for us as it allows us to ensure the hash is consistent.
func generateJobNameForSpec(spec *datatype.PluginSpec) (string, error) {
	specjson, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(specjson)
	instance := hex.EncodeToString(sum[:])[:8]
	parts := strings.Split(path.Base(spec.Image), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid plugin name %q", spec.Image)
	}
	return strings.Join([]string{parts[0], strings.ReplaceAll(parts[1], ".", "-"), instance}, "-"), nil
}

func (p *PluginCtl) Deploy(name string, selectorStr string, node string, privileged bool, pluginImage string, pluginArgs []string) (string, error) {
	selector, err := ParseSelector(selectorStr)
	if err != nil {
		return "", fmt.Errorf("Failed to parse selector %q", err.Error())
	}
	// split name:version from image string
	parts := strings.Split(path.Base(pluginImage), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid plugin image (plugin:version) %q", pluginImage)
	}
	pluginSpec := datatype.PluginSpec{
		Privileged: privileged,
		Node:       node,
		Image:      pluginImage,
		Args:       pluginArgs,
		Job:        "",
		Name:       name,
		Selector:   selector,
	}
	jobName, err := jobNameForSpec(&pluginSpec)
	if err != nil {
		return "", err
	}
	pluginSpec.Job = jobName
	plugin := &datatype.Plugin{
		Name:       jobName,
		Version:    parts[1],
		PluginSpec: pluginSpec,
	}
	job, err := p.ResourceManager.CreateJob(plugin)
	if err != nil {
		return "", err
	}
	deployedJob, err := p.ResourceManager.RunPlugin(job)
	return deployedJob.Name, err
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
				fmt.Println(string(buf[:numBytes]))
			}
		}
	}, flag, nil
}

func (p *PluginCtl) TerminatePlugin(pluginName string) error {
	return p.ResourceManager.TerminateJob(pluginName)
}

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
		status    v1.JobConditionType
	)
	for _, plugin := range list.Items {
		if strings.Contains(plugin.Name, "wes") {
			continue
		}
		startTime = ""
		duration = ""
		status = "UNKNOWN"
		if plugin.Status.StartTime != nil {
			startTime = plugin.Status.StartTime.Time.UTC().Format("2006/01/02 15:04:05 MST")
			if plugin.Status.CompletionTime != nil {
				duration = plugin.Status.CompletionTime.Sub(plugin.Status.StartTime.Time).String()
			}
		}
		if len(plugin.Status.Conditions) > 0 {
			status = plugin.Status.Conditions[0].Type
		}
		formattedList += fmt.Sprintf("%-*s%-*s%-*s%-*s\n", maxLengthName+3, plugin.Name, maxLengthStatus+3, status, maxLengthStartTime+3, startTime, maxLengthDuration+3, duration)
	}
	return
}
