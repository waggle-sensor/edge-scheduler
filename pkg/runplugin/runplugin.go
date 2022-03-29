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
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scheduler struct {
	KubernetesClientset *kubernetes.Clientset
	RabbitMQClient      *rabbithole.Client
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
	if err := updateKubernetesDeployment(sch.KubernetesClientset, config); err != nil {
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

var hostPathDirectoryOrCreate = apiv1.HostPathDirectoryOrCreate

func labelsForConfig(config *pluginConfig) map[string]string {
	return map[string]string{
		"app":                           config.Name,
		"role":                          "plugin", // TODO drop in place of sagecontinuum.org/role
		"sagecontinuum.org/role":        "plugin",
		"sagecontinuum.org/plugin-job":  config.Job,
		"sagecontinuum.org/plugin-task": config.Name,
	}
}

func nodeSelectorForConfig(config *pluginConfig) map[string]string {
	vals := map[string]string{}
	if config.Node != "" {
		vals["k3s.io/hostname"] = config.Node
	}
	for k, v := range config.Selector {
		vals[k] = v
	}
	return vals
}

func securityContextForConfig(config *pluginConfig) *apiv1.SecurityContext {
	if config.Privileged {
		return &apiv1.SecurityContext{Privileged: &config.Privileged}
	}
	return nil
}

func createDeploymentForConfig(config *pluginConfig) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": config.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsForConfig(config),
				},
				Spec: apiv1.PodSpec{
					NodeSelector: nodeSelectorForConfig(config),
					Containers: []apiv1.Container{
						{
							SecurityContext: securityContextForConfig(config),
							Name:            config.Name,
							Image:           config.Image,
							Args:            config.Args,
							Env: []apiv1.EnvVar{
								{
									Name:  "PULSE_SERVER",
									Value: "tcp:wes-audio-server:4713",
								},
								{
									Name:  "WAGGLE_PLUGIN_HOST",
									Value: "wes-rabbitmq",
								},
								{
									Name:  "WAGGLE_PLUGIN_PORT",
									Value: "5672",
								},
								{
									Name:  "WAGGLE_PLUGIN_USERNAME",
									Value: config.Username,
								},
								{
									Name:  "WAGGLE_PLUGIN_PASSWORD",
									Value: config.Password,
								},
								// NOTE WAGGLE_APP_ID is used to bind plugin <-> Pod identities.
								{
									Name: "WAGGLE_APP_ID",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.uid",
										},
									},
								},
								// Set pod IP for use by ROS clients.
								{
									Name: "ROS_IP",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								// Use default WES roscore hostname for ROS clients.
								{
									Name:  "ROS_MASTER_URI",
									Value: "http://wes-roscore.default.svc.cluster.local:11311",
								},
							},
							// NOTE This will provide WAGGLE_NODE_ID and WAGGLE_NODE_VSN for cases that a plugin
							// needs to make a node specific choice. This is not the ideal way to manage node
							// specific config, but may unblock things for now.
							EnvFrom: []apiv1.EnvFromSource{
								{
									ConfigMapRef: &apiv1.ConfigMapEnvSource{
										LocalObjectReference: apiv1.LocalObjectReference{
											Name: "wes-identity",
										},
									},
								},
							},
							Resources: apiv1.ResourceRequirements{
								Limits:   apiv1.ResourceList{},
								Requests: apiv1.ResourceList{},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "uploads",
									MountPath: "/run/waggle/uploads",
								},
								{
									Name:      "waggle-data-config",
									MountPath: "/run/waggle/data-config.json",
									SubPath:   "data-config.json",
								},
								{
									Name:      "wes-audio-server-plugin-conf",
									MountPath: "/etc/asound.conf",
									SubPath:   "asound.conf",
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "uploads",
							VolumeSource: apiv1.VolumeSource{
								HostPath: &apiv1.HostPathVolumeSource{
									Path: path.Join("/media/plugin-data/uploads", config.Name, config.Version),
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "waggle-data-config",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "waggle-data-config",
									},
								},
							},
						},
						{
							Name: "wes-audio-server-plugin-conf",
							VolumeSource: apiv1.VolumeSource{
								ConfigMap: &apiv1.ConfigMapVolumeSource{
									LocalObjectReference: apiv1.LocalObjectReference{
										Name: "wes-audio-server-plugin-conf",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func updateKubernetesDeployment(clientset *kubernetes.Clientset, config *pluginConfig) error {
	deployment := createDeploymentForConfig(config)

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
