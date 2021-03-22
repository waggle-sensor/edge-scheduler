package nodescheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ResourceManager structs a resource manager talking to a local computing cluster to schedule plugins
type ResourceManager struct {
	Namespace     string
	ECRRegistry   *url.URL
	ClientSet     *kubernetes.Clientset
	RMQManagement *RMQManagement
}

// NewResourceManager returns an instance of ResourceManager
func NewK3SResourceManager(namespace string, registry string, clientset *kubernetes.Clientset, rmqManagement *RMQManagement) (rm *ResourceManager, err error) {
	registryAddress, err := url.Parse(registry)
	if err != nil {
		return
	}
	rm = &ResourceManager{
		Namespace:     namespace,
		ECRRegistry:   registryAddress,
		ClientSet:     clientset,
		RMQManagement: rmqManagement,
	}
	return
}

func generatePassword() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// this should generally not fail. if it does, then we'll give up until the bigger error is resolved.
		panic(err)
	}
	return hex.EncodeToString(b)
}

// CreatePluginCredential creates a credential inside RabbitMQ server for the plugin
func (rm *ResourceManager) CreatePluginCredential(plugin *datatype.Plugin) (datatype.PluginCredential, error) {
	// TODO: We will need to add instance of plugin as a aprt of Username
	// username should follow "plugin.NAME:VERSION" format to publish messages via WES
	credential := datatype.PluginCredential{
		Username: fmt.Sprint("plugin.", strings.ToLower(plugin.Name), ":", plugin.Version),
		Password: generatePassword(),
	}
	return credential, nil
}

// CreateK3SDeployment creates and returns a K3S deployment object of the plugin
// It also embeds a K3S configmap for plugin if needed
func (rm *ResourceManager) CreateK3SDeployment(plugin *datatype.Plugin, credential datatype.PluginCredential) (*appsv1.Deployment, error) {
	// k3s does not accept uppercase letters as container name
	pluginNameInLowcase := strings.ToLower(plugin.Name)

	// Apply dataupload
	var hostPathDirectoryOrCreate = apiv1.HostPathDirectoryOrCreate
	specVolumes := []apiv1.Volume{
		{
			Name: "uploads",
			VolumeSource: apiv1.VolumeSource{
				HostPath: &apiv1.HostPathVolumeSource{
					Path: path.Join("/media/plugin-data/uploads", pluginNameInLowcase, plugin.Version),
					Type: &hostPathDirectoryOrCreate,
				},
			},
		},
	}
	containerVoumeMounts := []apiv1.VolumeMount{
		{
			Name:      "uploads",
			MountPath: "/run/waggle/uploads",
		},
	}

	// Apply datashim for the plugin if needed
	if plugin.DataShims != nil && len(plugin.DataShims) > 0 {
		configMapName := strings.ToLower("waggle-data-config-" + pluginNameInLowcase)
		err := rm.CreateDataConfigMap(configMapName, plugin.DataShims)
		if err != nil {
			return nil, err
		}
		// Create a volume for Spec
		var configMap apiv1.ConfigMapVolumeSource
		configMap.Name = configMapName
		volume := apiv1.Volume{
			Name: "waggle-data-config",
		}
		volume.ConfigMap = &configMap
		specVolumes = append(specVolumes, volume)
		// Create a volume mount for container
		containerVoumeMounts = append(containerVoumeMounts, apiv1.VolumeMount{
			Name:      "waggle-data-config",
			MountPath: "/run/waggle",
			SubPath:   "data-config.json",
		})
	}
	//TODO: Think about how to apply arguments and environments into k3s deployment
	//      This is related to performance related and unrelated knobs
	// if len(plugin.Args) > 0 {
	// 	container.Args = plugin.Args
	// }

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginNameInLowcase,
			Namespace: rm.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": pluginNameInLowcase,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  pluginNameInLowcase,
						"role": "plugin",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name: pluginNameInLowcase,
							Image: path.Join(
								rm.ECRRegistry.Path,
								strings.Join([]string{plugin.Name, plugin.Version}, ":"),
							),
							// Args: plugin.Args,
							Env: []apiv1.EnvVar{
								{
									Name:  "WAGGLE_PLUGIN_NAME",
									Value: strings.Join([]string{plugin.Name, plugin.Version}, ":"),
								},
								{
									Name:  "WAGGLE_PLUGIN_VERSION",
									Value: plugin.Version,
								},
								{
									Name:  "WAGGLE_PLUGIN_USERNAME",
									Value: credential.Username,
								},
								{
									Name:  "WAGGLE_PLUGIN_PASSWORD",
									Value: credential.Password,
								},
								{
									Name:  "WAGGLE_PLUGIN_HOST",
									Value: "rabbitmq-server",
								},
								{
									Name:  "WAGGLE_PLUGIN_PORT",
									Value: "5672",
								},
								// plugin.Envs..., TODO: if more envs need to be included
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
							VolumeMounts: containerVoumeMounts,
						},
					},
					Volumes: specVolumes,
				},
			},
		},
	}

	// d, _ := yaml.Marshal(&deployment)
	// fmt.Printf("--- t dump:\n%s\n\n", string(d))
	// fmt.Printf("%v", pod)

	return deployment, nil
}

// CreateDataConfigMap creates a K3S configmap object
func (rm *ResourceManager) CreateDataConfigMap(configName string, datashims []*datatype.DataShim) error {
	// Check if the configmap already exists
	configMaps, err := rm.ClientSet.CoreV1().ConfigMaps(rm.Namespace).List(context.TODO(), metav1.ListOptions{})
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
	_, err = rm.ClientSet.CoreV1().ConfigMaps(rm.Namespace).Create(context.TODO(), &config, metav1.CreateOptions{})
	return err
}

// LaunchPlugin launches a k3s deployment in the cluster
func (rm *ResourceManager) LaunchPlugin(deployment *appsv1.Deployment) error {
	deploymentsClient := rm.ClientSet.AppsV1().Deployments(rm.Namespace)

	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		logger.Info.Printf("Failed to create deployment %s.\n", err)
	}
	logger.Info.Printf("Created deployment for plugin %q\n", result.GetObjectMeta().GetName())
	return err
}

// TerminatePlugin terminates the k3s deployment of given plugin name
func (rm *ResourceManager) TerminatePlugin(pluginName string) error {
	pluginNameInLowcase := strings.ToLower(pluginName)

	deploymentsClient := rm.ClientSet.AppsV1().Deployments(rm.Namespace)
	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	exists := false
	for _, d := range list.Items {
		if d.Name == pluginNameInLowcase {
			exists = true
			break
		}
	}
	if !exists {
		logger.Error.Printf("Could not terminate plugin %s: not exist", pluginNameInLowcase)
		return nil
	}

	// fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(context.TODO(), pluginNameInLowcase, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return err
	}
	logger.Info.Printf("Deleted deployment of plugin %s", pluginNameInLowcase)
	return err
}

func (rm *ResourceManager) Run(chanPluginToUpdate <-chan *datatype.Plugin) {
	for {
		plugin := <-chanPluginToUpdate
		logger.Debug.Printf("Plugin %s:%s needs to be in %s state", plugin.Name, plugin.Version, plugin.Status.SchedulingStatus)
		if plugin.Status.SchedulingStatus == datatype.Running {
			credential, err := rm.CreatePluginCredential(plugin)
			if err != nil {
				logger.Error.Printf("Could not create a plugin credential for %s on RabbitMQ at %s: %s", plugin.Name, rm.RMQManagement.client.Endpoint, err.Error())
				continue
			}
			err = rm.RMQManagement.RegisterPluginCredential(credential)
			if err != nil {
				logger.Error.Printf("Could not register the credential %s to RabbitMQ at %s: %s", credential.Username, rm.RMQManagement.client.Endpoint, err.Error())
				continue
			}
			deployablePlugin, err := rm.CreateK3SDeployment(plugin, credential)
			if err != nil {
				logger.Error.Printf("Could not create a k3s deployment for plugin %s: %s", plugin.Name, err.Error())
				continue
			}
			err = rm.LaunchPlugin(deployablePlugin)
			if err != nil {
				logger.Error.Printf("Failed to launch plugin %s: %s", plugin.Name, err.Error())
			}
		} else if plugin.Status.SchedulingStatus == datatype.Stopped {
			err := rm.TerminatePlugin(plugin.Name)
			if err != nil {
				logger.Error.Printf("Failed to stop plugin %s: %s", plugin.Name, err.Error())
			}
		}
	}
}

// RMQManagement structs a connection to RMQManagement
type RMQManagement struct {
	rabbitmqManagementURI      string
	rabbitmqManagementUsername string
	rabbitmqManagementPassword string
	client                     *rabbithole.Client
}

// NewRMQManagement creates and returns an instance of connection to RMQManagement
func NewRMQManagement(rmqManagementURI string, rmqManagementUsername string, rmqManagementPassword string) (*RMQManagement, error) {
	c, err := rabbithole.NewClient(rmqManagementURI, rmqManagementUsername, rmqManagementPassword)
	if err != nil {
		return nil, err
	}
	return &RMQManagement{
		rabbitmqManagementURI:      rmqManagementURI,
		rabbitmqManagementUsername: rmqManagementUsername,
		rabbitmqManagementPassword: rmqManagementPassword,
		client:                     c,
	}, nil
}

// RegisterPluginCredential registers given plugin credential to designated RMQ server
func (rmq *RMQManagement) RegisterPluginCredential(credential datatype.PluginCredential) error {
	// The functions below come from Sean's RunPlugin
	if _, err := rmq.client.PutUser(credential.Username, rabbithole.UserSettings{
		Password: credential.Password,
	}); err != nil {
		return err
	}

	if _, err := rmq.client.UpdatePermissionsIn("/", credential.Username, rabbithole.Permissions{
		Configure: "^amq.gen",
		Read:      ".*",
		Write:     ".*",
	}); err != nil {
		return err
	}
	logger.Debug.Printf("Plugin credential %s:%s is registered in RabbitMQ at %s", credential.Username, credential.Password, rmq.rabbitmqManagementURI)
	return nil
}

func int32Ptr(i int32) *int32 { return &i }

// GetK3SClient returns an instance of clientset talking to a K3S cluster
func GetK3SClient(incluster bool, pathToConfig string) (*kubernetes.Clientset, error) {
	if incluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", pathToConfig)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}
}
