package runplugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/nodescheduler"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scheduler struct {
	KubernetesClientset *kubernetes.Clientset
	RabbitMQClient      *rabbithole.Client
}

// type Spec struct {
// 	Image      string            `json:"image"`
// 	Args       []string          `json:"args"`
// 	Privileged bool              `json:"privileged"`
// 	Node       string            `json:"node"`
// 	Job        string            `json:"job"`
// 	Name       string            `json:"name"`
// 	Selector   map[string]string `json:"selector"`
// }

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

// RunPlugin prepares to run a plugin image
// TODO wrap k8s and rmq clients into single config struct
func (sch *Scheduler) RunPlugin(plugin *datatype.Plugin) error {
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

	log.Printf("updating kubernetes deployment for %q", spec.Image)
	if err := updateKubernetesDeployment(sch.KubernetesClientset, plugin); err != nil {
		return err
	}

	log.Printf("plugin ready %q", spec.Image)

	return nil
}

func createDeploymentForConfig(plugin *datatype.Plugin) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: plugin.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": plugin.Name,
				},
			},
			Template: nodescheduler.CreatePodTemplateSpecForPlugin(plugin),
		},
	}
}

func updateKubernetesDeployment(clientset *kubernetes.Clientset, plugin *datatype.Plugin) error {
	deployment := createDeploymentForConfig(plugin)

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
