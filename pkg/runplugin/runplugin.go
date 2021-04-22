package runplugin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"path"
	"strings"

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

// RunPlugin prepares to run a plugin image
// TODO wrap k8s and rmq clients into single config struct
func (sch *Scheduler) RunPlugin(image string, args ...string) error {
	base := path.Base(image)

	// split name:version from image string
	parts := strings.Split(base, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid plugin name %q", image)
	}

	config := &pluginConfig{
		Image:   image,
		Name:    parts[0],
		Version: parts[1],
		// NOTE(sean) username will be validated by wes-data-sharing-service. see: https://github.com/waggle-sensor/wes-data-sharing-service/blob/0e5a44b1ce6e6109a660b2922f56523099054750/main.py#L34
		Username: "plugin." + base,
		Password: generatePassword(),
		Args:     args,
	}

	log.Printf("setting up plugin %q", image)

	log.Printf("creating rabbitmq plugin user %q for %q", config.Username, image)
	if err := createRabbitmqUser(sch.RabbitMQClient, config); err != nil {
		return err
	}

	log.Printf("creating kubernetes deployment for %q", image)
	if err := createKubernetesDeployment(sch.KubernetesClientset, config); err != nil {
		return err
	}

	log.Printf("plugin ready %q", image)

	return nil
}

type pluginConfig struct {
	Image    string
	Name     string
	Version  string
	Username string
	Password string
	Args     []string
}

var hostPathDirectoryOrCreate = apiv1.HostPathDirectoryOrCreate

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
					Labels: map[string]string{
						"app":  config.Name,
						"role": "plugin",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  config.Name,
							Image: config.Image,
							Args:  config.Args,
							Env: []apiv1.EnvVar{
								{
									Name:  "WAGGLE_PLUGIN_NAME",
									Value: config.Name + ":" + config.Version,
								},
								{
									Name:  "WAGGLE_PLUGIN_VERSION",
									Value: config.Version,
								},
								{
									Name:  "WAGGLE_PLUGIN_USERNAME",
									Value: config.Username,
								},
								{
									Name:  "WAGGLE_PLUGIN_PASSWORD",
									Value: config.Password,
								},
								{
									Name:  "WAGGLE_PLUGIN_HOST",
									Value: "wes-rabbitmq",
								},
								{
									Name:  "WAGGLE_PLUGIN_PORT",
									Value: "5672",
								},
							},
							EnvFrom: []apiv1.EnvFromSource{
								{
									ConfigMapRef: &apiv1.ConfigMapEnvSource{
										LocalObjectReference: apiv1.LocalObjectReference{
											Name: "waggle-config",
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
					},
				},
			},
		},
	}
}

func generatePassword() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// this should generally not fail. if it does, then we'll give up until the bigger error is resolved.
		panic(err)
	}
	return hex.EncodeToString(b)
}

func createKubernetesDeployment(clientset *kubernetes.Clientset, config *pluginConfig) error {
	deployment := createDeploymentForConfig(config)
	// ensure existing deployments are deleted
	clientset.AppsV1().Deployments("default").Delete(context.TODO(), deployment.ObjectMeta.Name, metav1.DeleteOptions{})
	// create new deployment
	if _, err := clientset.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func createRabbitmqUser(rmqclient *rabbithole.Client, config *pluginConfig) error {
	if _, err := rmqclient.PutUser(config.Username, rabbithole.UserSettings{
		Password: config.Password,
	}); err != nil {
		return err
	}

	if _, err := rmqclient.UpdatePermissionsIn("/", config.Username, rabbithole.Permissions{
		Configure: "^amq.gen",
		Read:      ".*",
		Write:     ".*",
	}); err != nil {
		return err
	}

	return nil
}
