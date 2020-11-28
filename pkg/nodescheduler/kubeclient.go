package nodescheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace    = "sage-development"
	ECRRegistry  *url.URL
	clientSet    *kubernetes.Clientset
	emulatingK3S = false
)

// InitializeK3S loads k3s configuration to talk to k3s cluster
func InitializeK3S(chanPluginToUpdate <-chan *datatype.Plugin, registryAddress string, emulating bool) {
	if !emulating {
		configPath := "/root/.kube/config"
		cs, err := getClient(configPath)
		if err != nil {
			logger.Error.Printf("Could not read kubeconfig at %s", configPath)
			logger.Info.Printf("k3sclient runs on emulation mode")
			emulating = true
		}
		clientSet = cs
	}
	ECRRegistry, _ = url.Parse(registryAddress)
	emulatingK3S = emulating
	go runK3SClient(chanPluginToUpdate)
}

func runK3SClient(chanPluginToUpdate <-chan *datatype.Plugin) {
	for {
		plugin := <-chanPluginToUpdate
		_, err := CreateK3SDeployment(plugin)
		if err != nil {
			logger.Error.Printf("Could not create k3s deployment for plugin %s", plugin.Name)
			continue
		}
		// if plugin.Status.SchedulingStatus == datatype.Running {
		// 	err = LaunchPlugin(k3sDeployment)
		// }
	}
}

func CreateK3SDeployment(plugin *datatype.Plugin) (pod *appsv1.Deployment, err error) {
	pod = &appsv1.Deployment{}

	var specVolumes []apiv1.Volume
	var containerVoumeMounts []apiv1.VolumeMount
	var container apiv1.Container
	container.Name = strings.ToLower(plugin.Name)
	container.Image = path.Join(
		ECRRegistry.Path,
		strings.Join([]string{plugin.Name, plugin.Version}, ":"),
	)
	// Apply datashim for the plugin
	if plugin.DataShims != nil && len(plugin.DataShims) > 0 {
		configMapName := strings.ToLower("datashim-" + plugin.Name)
		if !emulatingK3S {
			err := createDataConfigMap(configMapName, plugin.DataShims)
			if err != nil {
				return nil, err
			}
		}
		// Create a volume for Spec
		var configMap apiv1.ConfigMapVolumeSource
		configMap.Name = configMapName
		volume := apiv1.Volume{
			Name: "datashim",
		}
		volume.ConfigMap = &configMap
		specVolumes = append(specVolumes, volume)
		// Create a volume mount for container
		containerVoumeMounts = append(containerVoumeMounts, apiv1.VolumeMount{
			Name:      "datashim",
			MountPath: "/run/waggle",
		})
	}
	//TODO: Think about how to apply arguments and environments into k3s deployment
	//      This is related to performance related and unrelated knobs
	// if len(plugin.Args) > 0 {
	// 	container.Args = plugin.Args
	// }
	// if len(plugin.Env) > 0 {
	// 	var envs []apiv1.EnvVar
	// 	for k, v := range plugin.Env {
	// 		var env apiv1.EnvVar
	// 		env.Name = k
	// 		env.Value = v
	// 		envs = append(envs, env)
	// 	}
	// 	container.Env = envs
	// }

	container.VolumeMounts = containerVoumeMounts

	// Set plugin name and namespace
	pod.ObjectMeta = metav1.ObjectMeta{
		Name:      strings.ToLower(plugin.Name),
		Namespace: namespace,
	}

	pod.Spec = appsv1.DeploymentSpec{
		Replicas: int32Ptr(1),
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": strings.ToLower(plugin.Name),
			},
		},
		Template: apiv1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": strings.ToLower(plugin.Name),
				},
			},
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{container},
				Volumes:    specVolumes,
			},
		},
	}
	d, _ := yaml.Marshal(&pod)
	fmt.Printf("--- t dump:\n%s\n\n", string(d))
	fmt.Printf("%v", pod)
	return
}

// LaunchPlugin launches a k3s deployment in the cluster
func LaunchPlugin(deployment *appsv1.Deployment) error {
	deploymentsClient := clientSet.AppsV1().Deployments(namespace)

	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		logger.Info.Printf("Failed to create deployment %s.\n", err)
	}
	logger.Info.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
	return err
}

// TerminatePlugin terminates the k3s deployment of given plugin name
func TerminatePlugin(plugin string) error {
	plugin = strings.ToLower(plugin)

	deploymentsClient := clientSet.AppsV1().Deployments(namespace)
	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	exists := false
	for _, d := range list.Items {
		if d.Name == plugin {
			exists = true
			break
		}
	}
	if !exists {
		logger.Error.Printf("Could not terminate plugin %s: not exist", plugin)
		return nil
	}

	// fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), plugin, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return err
	}
	logger.Info.Printf("Deleted deployment %s.\n", plugin)
	return err
}

// func CreateK3sPod(pluginConfig PluginConfig) *appsv1.Deployment {
// 	var pod appsv1.Deployment
// 	var volumes []apiv1.Volume
//
// 	// Build containers
// 	var containers []apiv1.Container
// 	for _, plugin := range pluginConfig.Plugins {
// 		var container apiv1.Container
// 		container.Name = strings.ToLower(plugin.Name)
// 		container.Image = plugin.Image
// 		if len(plugin.Args) > 0 {
// 			container.Args = plugin.Args
// 		}
// 		if len(plugin.Env) > 0 {
// 			var envs []apiv1.EnvVar
// 			for k, v := range plugin.Env {
// 				var env apiv1.EnvVar
// 				env.Name = k
// 				env.Value = v
// 				envs = append(envs, env)
// 			}
// 			container.Env = envs
// 		}
//
// 		// Configure data-shim
// 		if value, ok := plugin.Configs["dataConfig"]; ok {
// 			var configMapName = strings.ToLower(pluginConfig.Name + "-" + plugin.Name)
// 			err := createDataConfigMap(configMapName, value)
// 			if err != nil {
// 				panic(err.Error())
// 			}
// 			// Create a volume for Spec
// 			var volume apiv1.Volume
// 			var configMap apiv1.ConfigMapVolumeSource
// 			configMap.Name = configMapName
// 			volume.Name = "data-shim"
// 			volume.ConfigMap = &configMap
// 			// volume.ConfigMap = &apiv1.ConfigMapVolumeSource{
// 			// 	Name: configMapName,
// 			// }
// 			volumes = append(volumes, volume)
//
// 			// Create a volume mount for container
// 			container.VolumeMounts = []apiv1.VolumeMount{
// 				{
// 					Name:      "data-shim",
// 					MountPath: "/run/waggle",
// 				},
// 			}
// 		}
// 		containers = append(containers, container)
// 	}
//
// 	// Set plugin name and namespace
// 	pod.ObjectMeta = metav1.ObjectMeta{
// 		Name:      strings.ToLower(pluginConfig.Name),
// 		Namespace: namespace,
// 	}
//
// 	pod.Spec = appsv1.DeploymentSpec{
// 		Replicas: int32Ptr(1),
// 		Selector: &metav1.LabelSelector{
// 			MatchLabels: map[string]string{
// 				"app": strings.ToLower(pluginConfig.Name),
// 			},
// 		},
// 		Template: apiv1.PodTemplateSpec{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Labels: map[string]string{
// 					"app": strings.ToLower(pluginConfig.Name),
// 				},
// 			},
// 			Spec: apiv1.PodSpec{
// 				Containers: containers,
// 				Volumes:    volumes,
// 			},
// 		},
// 	}
// 	d, _ := yaml.Marshal(&pod)
// 	fmt.Printf("--- t dump:\n%s\n\n", string(d))
// 	fmt.Printf("%v", pod)
// 	return &pod
// }

func int32Ptr(i int32) *int32 { return &i }

func getClient(pathToConfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", pathToConfig)
	if err != nil {
		panic(err.Error())
	}
	return kubernetes.NewForConfig(config)
}

func createDataConfigMap(configName string, datashims []*datatype.DataShim) (err error) {
	// Check if the configmap already exists
	configMaps, err := clientSet.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, c := range configMaps.Items {
		if c.Name == configName {
			// TODO: May want to renew the existing one
			logger.Info.Printf("ConfigMap %s already exists", configName)
			return nil
		}
	}
	data, err := json.Marshal(datashims)
	if err != nil {
		return err
	}
	var config apiv1.ConfigMap
	config.Name = configName
	config.Data = make(map[string]string)
	config.Data["data-config.json"] = string(data)
	_, err = clientSet.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &config, metav1.CreateOptions{})
	return err
}
