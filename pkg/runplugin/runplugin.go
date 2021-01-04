package runplugin

import (
	"context"
	"fmt"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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
									Value: "rabbitmq-server",
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

// RunPlugin prepares to run a plugin image
func RunPlugin(clientset *kubernetes.Clientset, image string, args ...string) error {
	base := path.Base(image)

	// split name:version from image string
	parts := strings.Split(base, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid plugin name %q", image)
	}

	deployment := createDeploymentForConfig(&pluginConfig{
		Image:    image,
		Name:     parts[0],
		Version:  parts[1],
		Username: strings.ReplaceAll(base, ":", "-"),
		Password: "averysecurepassword",
		Args:     args,
	})

	// ensure existing deployments are deleted
	clientset.AppsV1().Deployments("default").Delete(context.TODO(), deployment.ObjectMeta.Name, metav1.DeleteOptions{})

	if _, err := clientset.AppsV1().Deployments("default").Create(context.TODO(), deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
