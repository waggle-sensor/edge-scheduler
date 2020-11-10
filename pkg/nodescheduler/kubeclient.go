package nodescheduler

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	namespace string
	clientSet *kubernetes.Clientset
)

func InitializeK3s() {
	var err error
	clientSet, err = getClient("/root/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	namespace = "sage-development"
}

func LaunchPlugin(deployment *appsv1.Deployment) bool {
	deploymentsClient := clientSet.AppsV1().Deployments(namespace)

	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
	return true
}

func TerminatePlugin(plugin string) bool {
	plugin = strings.ToLower(plugin)
	var kubeconfig string
	kubeconfig = "/root/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	deploymentsClient := clientset.AppsV1().Deployments(namespace)
	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, d := range list.Items {
		fmt.Printf(" * %s (%d replicas)\n", d.Name, *d.Spec.Replicas)
	}

	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), plugin, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Printf("Deleted deployment %s.\n", plugin)
	return true
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

func createDataConfigMap(configName string, configPath string) (err error) {
	// Check if the configmap already exists
	configMaps, err := clientSet.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, c := range configMaps.Items {
		if c.Name == configName {
			// TODO: May want to renew the existing one
			fmt.Println("ConfigMap already exists")
			return nil
		}
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return
	}
	fmt.Println(string(data))
	var config apiv1.ConfigMap
	config.Name = configName
	config.Data = make(map[string]string)
	config.Data["data-config.json"] = string(data)
	_, err = clientSet.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &config, metav1.CreateOptions{})
	if err != nil {
		return
	}
	return nil
}
