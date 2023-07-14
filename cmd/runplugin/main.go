package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
)

func getenv(key string, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

const rancherKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"

func detectDefaultKubeconfig() string {
	if _, err := os.ReadFile(rancherKubeconfigPath); err == nil {
		return rancherKubeconfigPath
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}

func main() {
	var (
		privileged  bool
		job         string
		name        string
		node        string
		selectorStr string
		kubeconfig  string
	)

	flag.BoolVar(&privileged, "privileged", false, "run as privileged plugin")
	flag.StringVar(&job, "job", "sage", "specify plugin job")
	flag.StringVar(&name, "name", "", "specify plugin name")
	flag.StringVar(&node, "node", "", "run plugin on node")
	flag.StringVar(&selectorStr, "selector", "", "selector specifying where plugin can run")
	flag.StringVar(&kubeconfig, "kubeconfig", getenv("KUBECONFIG", detectDefaultKubeconfig()), "path to the kubeconfig file")
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		fmt.Printf("%s plugin-image [plugin-args]", os.Args[0])
		os.Exit(1)
	}

	resourceManager, err := nodescheduler.NewK3SResourceManager(false, kubeconfig, "runplugin", false)
	if err != nil {
		log.Fatalf("nodescheduler.NewK3SResourceManager: %s", err.Error())
	}
	resourceManager.Namespace = "default"

	args := flag.Args()

	selector, err := parseSelector(selectorStr)
	if err != nil {
		log.Fatalf("parseSelector: %s", err.Error())
	}

	plugin := &datatype.Plugin{
		Name: name,
		PluginSpec: &datatype.PluginSpec{
			Privileged: privileged,
			Node:       node,
			Image:      args[0],
			Args:       args[1:],
			Job:        job,
			Selector:   selector,
		},
	}

	if err := runPlugin(resourceManager, plugin); err != nil {
		log.Fatalf("runPlugin: %s", err.Error())
	}
}

var validNamePattern = regexp.MustCompile("^[a-z0-9-]+$")

// generatePluginNameForSpec generates a consistent name for a Spec.
//
// Very important note from: https://pkg.go.dev/encoding/json#Marshal
//
// Map values encode as JSON objects. The map's key type must either be a string, an integer type,
// or implement encoding.TextMarshaler. The map keys are sorted and used as JSON object keys by applying
// the following rules, subject to the UTF-8 coercion described for string values above:
//
// The "map keys are sorted" bit is important for us as it allows us to ensure the hash is consistent.
func generatePluginNameForSpec(spec *datatype.PluginSpec) (string, error) {
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

// runPlugin prepares to run a plugin image
// TODO wrap k8s and rmq clients into single config struct
func runPlugin(resourceManager *nodescheduler.ResourceManager, plugin *datatype.Plugin) error {
	spec := plugin.PluginSpec
	// split name:version from image string
	parts := strings.Split(path.Base(spec.Image), ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid plugin name %q", spec.Image)
	}

	// generate name if given name is empty
	if plugin.Name == "" {
		name, err := generatePluginNameForSpec(plugin.PluginSpec)
		if err != nil {
			return fmt.Errorf("failed to generate name: %s", err.Error())
		}
		plugin.Name = name
	}

	// validate plugin name
	if !validNamePattern.MatchString(plugin.Name) {
		return fmt.Errorf("plugin name must consist of alphanumeric characters with '-' RFC1123")
	}

	log.Printf("plugin name is %s", plugin.Name)

	deployment, err := resourceManager.CreateDeploymentTemplate(&datatype.PluginRuntime{
		Plugin: *plugin,
	})
	if err != nil {
		return fmt.Errorf("resourceManager.CreateDeployment: %s", err.Error())
	}

	log.Printf("updating kubernetes deployment for %q", spec.Image)
	if err := updateDeployment(resourceManager.Clientset, deployment); err != nil {
		return fmt.Errorf("updateDeployment: %s", err.Error())
	}

	log.Printf("plugin ready %q", spec.Image)

	return nil
}

func updateDeployment(clientset *kubernetes.Clientset, deployment *v1.Deployment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployments := clientset.AppsV1().Deployments("default")

	// if deployment exists, then update it, else create it
	if _, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{}); err == nil {
		_, err := deployments.Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	} else {
		_, err := deployments.Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}
}
