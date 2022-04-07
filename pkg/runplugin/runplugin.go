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
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scheduler struct {
	KubernetesClientset *kubernetes.Clientset
	RabbitMQClient      *rabbithole.Client
	ResourceManager     *nodescheduler.ResourceManager
}

type Spec struct {
	Image      string            `json:"image"`
	Args       []string          `json:"args"`
	Privileged bool              `json:"privileged"`
	Node       string            `json:"node"`
	Job        string            `json:"job"`
	Name       string            `json:"name"`
	Selector   map[string]string `json:"selector"`
}

var validNamePattern = regexp.MustCompile("^[a-z0-9-]+$")

func pluginNameForSpec(spec *Spec) (string, error) {
	// if no given name for the plugin, use PLUGIN-VERSION-INSTANCE format for name
	// INSTANCE is calculated as Sha256("DOMAIN/PLUGIN:VERSION&ARGUMENTS") and
	// take the first 8 hex letters.
	// NOTE: if multiple plugins with the same version and arguments are given for
	//       the same domain, only one deployment will be applied to the cluster
	// NOTE2: To comply with RFC 1123 for Kubernetes object name, only lower alphanumeric
	//        characters with '-' is allowed
	if spec.Name != "" {
		if !validNamePattern.MatchString(spec.Name) {
			return "", fmt.Errorf("plugin name must consist of alphanumeric characters with '-' RFC1123")
		}
		return spec.Name, nil
	}
	return generatePluginNameForSpec(spec)
}

// generatePluginNameForSpec generates a consistent name for a Spec.
//
// Very important note from: https://pkg.go.dev/encoding/json#Marshal
//
// Map values encode as JSON objects. The map's key type must either be a string, an integer type,
// or implement encoding.TextMarshaler. The map keys are sorted and used as JSON object keys by applying
// the following rules, subject to the UTF-8 coercion described for string values above:
//
// The "map keys are sorted" bit is important for us as it allows us to ensure the hash is consistent.
func generatePluginNameForSpec(spec *Spec) (string, error) {
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
func (sch *Scheduler) RunPlugin(spec *Spec) error {
	// split name:version from image string
	parts := strings.Split(path.Base(spec.Image), ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid plugin name %q", spec.Image)
	}

	pluginName, err := pluginNameForSpec(spec)
	if err != nil {
		return err
	}
	log.Printf("plugin name is %s", pluginName)

	config := &pluginConfig{
		Spec:    spec,
		Name:    pluginName,
		Version: parts[1],
		// NOTE(sean) username will be validated by wes-data-sharing-service. see: https://github.com/waggle-sensor/wes-data-sharing-service/blob/0e5a44b1ce6e6109a660b2922f56523099054750/main.py#L34
		Username: "plugin",
		Password: "plugin",
	}

	log.Printf("updating kubernetes deployment for %q", spec.Image)
	if err := updateKubernetesDeployment(sch.KubernetesClientset, sch.ResourceManager, config); err != nil {
		return err
	}

	log.Printf("plugin ready %q", spec.Image)

	return nil
}

type pluginConfig struct {
	*Spec
	Name     string
	Version  string
	Username string
	Password string
}

func updateKubernetesDeployment(clientset *kubernetes.Clientset, rm *nodescheduler.ResourceManager, config *pluginConfig) error {
	deployment, err := rm.NewPluginDeployment(&datatype.Plugin{
		Name: config.Name,
		PluginSpec: &datatype.PluginSpec{
			Image:      config.Image,
			Args:       config.Args,
			Privileged: config.Privileged,
			Node:       config.Node,
			Job:        config.Job,
			Selector:   config.Selector,
		},
	})
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deployments := clientset.AppsV1().Deployments(rm.Namespace)

	// if deployment exists, then update it, else create it
	if _, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{}); err == nil {
		_, err := deployments.Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	} else {
		_, err := deployments.Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}
}
