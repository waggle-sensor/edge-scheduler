package nodescheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	namespace             = "ses"
	rancherKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
)

var (
	hostPathDirectoryOrCreate       = apiv1.HostPathDirectoryOrCreate
	backOffLimit              int32 = 0
)

// ResourceManager structs a resource manager talking to a local computing cluster to schedule plugins
type ResourceManager struct {
	Namespace     string
	ECRRegistry   *url.URL
	Clientset     *kubernetes.Clientset
	RMQManagement *RMQManagement
	Simulate      bool
}

// NewResourceManager returns an instance of ResourceManager
func NewK3SResourceManager(registry string, incluster bool, kubeconfig string, rmqManagement *RMQManagement, simulate bool) (rm *ResourceManager, err error) {
	if simulate {
		return &ResourceManager{
			Namespace:     namespace,
			ECRRegistry:   nil,
			Clientset:     nil,
			RMQManagement: rmqManagement,
			Simulate:      simulate,
		}, nil
	}
	registryAddress, err := url.Parse(registry)
	if err != nil {
		return
	}
	k3sClient, err := GetK3SClient(incluster, kubeconfig)
	if err != nil {
		return
	}
	return &ResourceManager{
		Namespace:     namespace,
		ECRRegistry:   registryAddress,
		Clientset:     k3sClient,
		RMQManagement: rmqManagement,
		Simulate:      simulate,
	}, nil
}

func generatePassword() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// this should generally not fail. if it does, then we'll give up until the bigger error is resolved.
		panic(err)
	}
	return hex.EncodeToString(b)
}

func labelsForConfig(plugin *datatype.Plugin) map[string]string {
	return map[string]string{
		"app":                           plugin.Name,
		"role":                          "plugin", // TODO drop in place of sagecontinuum.org/role
		"sagecontinuum.org/role":        "plugin",
		"sagecontinuum.org/plugin-job":  plugin.PluginSpec.Job,
		"sagecontinuum.org/plugin-task": plugin.Name,
	}
}

func nodeSelectorForConfig(plugin *datatype.Plugin) map[string]string {
	vals := map[string]string{}
	if plugin.PluginSpec.Node != "" {
		vals["k3s.io/hostname"] = plugin.PluginSpec.Node
	}
	for k, v := range plugin.PluginSpec.Selector {
		vals[k] = v
	}
	return vals
}

func securityContextForConfig(plugin *datatype.Plugin) *apiv1.SecurityContext {
	if plugin.PluginSpec.Privileged {
		return &apiv1.SecurityContext{Privileged: &plugin.PluginSpec.Privileged}
	}
	return nil
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

// CreateNamespace creates a Kubernetes namespace
//
// If the namespace exists, it does nothing
func (rm *ResourceManager) CreateNamespace(namespace string) error {
	logger.Debug.Printf("Creating namespace %s", namespace)
	_, err := rm.Clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug.Printf("The namespace %s does not exist. Will create...", namespace)
		} else {
			logger.Debug.Printf("Failed to get %s from cluster: %s", namespace, err.Error())
			return err
		}
	} else {
		logger.Debug.Printf("The namespace %s already exists.", namespace)
		return nil
	}
	objNamespace := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = rm.Clientset.CoreV1().Namespaces().Create(
		context.TODO(),
		objNamespace,
		metav1.CreateOptions{},
	)
	return err
}

// ForwardService forwards a service from one namespace to other namespace for given ports
//
// This is useful when pods in the other namespace need to access to the service
func (rm *ResourceManager) ForwardService(serviceName string, fromNamespace string, toNamespace string) error {
	logger.Debug.Printf("Forwarding service %s from %s namespace to %s namespace", serviceName, fromNamespace, toNamespace)
	existingServiceInFromNamespace, err := rm.Clientset.CoreV1().Services(fromNamespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug.Printf("The service %s does not exist in %s namespace ", serviceName, fromNamespace)
			return err
		}
		logger.Debug.Printf("Failed to get %s from namespace %s: %s", serviceName, fromNamespace, err.Error())
		return err
	}
	objService := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: toNamespace,
		},
		Spec: apiv1.ServiceSpec{
			Type:         apiv1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, fromNamespace),
			Ports:        existingServiceInFromNamespace.Spec.Ports,
		},
	}
	existingServiceInToNamespace, err := rm.Clientset.CoreV1().Services(toNamespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug.Printf("The service %s does not exist in the namespace %s. Will create...", serviceName, toNamespace)
			_, err = rm.Clientset.CoreV1().Services(toNamespace).Create(context.TODO(), objService, metav1.CreateOptions{})
			return err
		}
		logger.Debug.Printf("Failed to get %s from namespace %s: %s", serviceName, toNamespace, err.Error())
		return err
	}
	objService.ObjectMeta.ResourceVersion = existingServiceInToNamespace.ObjectMeta.ResourceVersion
	_, err = rm.Clientset.CoreV1().Services(toNamespace).Update(context.TODO(), objService, metav1.UpdateOptions{})
	return err
}

func (rm *ResourceManager) CreateConfigMap(name string, data map[string]string, namespace string) error {
	var config apiv1.ConfigMap
	config.Name = name
	config.Data = data
	if namespace == "" {
		namespace = rm.Namespace
	}
	_, err := rm.Clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = rm.Clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), &config, metav1.CreateOptions{})
			return err
		} else {
			return err
		}
	}
	_, err = rm.Clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), &config, metav1.UpdateOptions{})
	return err
}

// WatchConfigMap
func (rm *ResourceManager) WatchConfigMap(name string, namespace string) (watch.Interface, error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	configMap, err := rm.Clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// var selector *metav1.LabelSelector
	// err = metav1.Convert_Map_string_To_string_To_v1_LabelSelector(&configMap.Labels, selector, nil)
	watcher, err := rm.Clientset.CoreV1().ConfigMaps(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: configMap.Name, Namespace: configMap.Namespace}))
	return watcher, err
}

func (rm *ResourceManager) WatchJob(name string, namespace string) (watch.Interface, error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	job, err := rm.Clientset.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// var selector *metav1.LabelSelector
	// err = metav1.Convert_Map_string_To_string_To_v1_LabelSelector(&configMap.Labels, selector, nil)
	watcher, err := rm.Clientset.BatchV1().Jobs(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: job.Name, Namespace: job.Namespace}))
	return watcher, err
}

// CreateK3SJob creates and returns a Kubernetes job object of the pllugin
func (rm *ResourceManager) CreateJob(plugin *datatype.Plugin) (*batchv1.Job, error) {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      plugin.Name,
			Namespace: rm.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelsForConfig(plugin),
				},
				Spec: apiv1.PodSpec{
					NodeSelector:  nodeSelectorForConfig(plugin),
					RestartPolicy: apiv1.RestartPolicyNever,
					Containers: []apiv1.Container{
						{
							SecurityContext: securityContextForConfig(plugin),
							Name:            plugin.Name,
							Image:           plugin.PluginSpec.Image,
							Args:            plugin.PluginSpec.Args,
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
									Value: "plugin",
								},
								{
									Name:  "WAGGLE_PLUGIN_PASSWORD",
									Value: "plugin",
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
									Path: path.Join("/media/plugin-data/uploads", plugin.PluginSpec.Name, plugin.Version),
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
			BackoffLimit: &backOffLimit,
		},
	}, nil
}

// CreateDeployment creates and returns a Kubernetes deployment object of the plugin
// It also embeds a K3S configmap for plugin if needed
func (rm *ResourceManager) CreateDeployment(plugin *datatype.Plugin, credential datatype.PluginCredential) (*appsv1.Deployment, error) {
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
									Value: "wes-rabbitmq",
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
	configMaps, err := rm.Clientset.CoreV1().ConfigMaps(rm.Namespace).List(context.TODO(), metav1.ListOptions{})
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
	_, err = rm.Clientset.CoreV1().ConfigMaps(rm.Namespace).Create(context.TODO(), &config, metav1.CreateOptions{})
	return err
}

func (rm *ResourceManager) RunPlugin(job *batchv1.Job) (*batchv1.Job, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	jobClient := rm.Clientset.BatchV1().Jobs(rm.Namespace)
	return jobClient.Create(ctx, job, metav1.CreateOptions{})
}

// LaunchPlugin launches a k3s deployment in the cluster
func (rm *ResourceManager) LaunchPlugin(deployment *appsv1.Deployment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	deploymentsClient := rm.Clientset.AppsV1().Deployments(rm.Namespace)
	result, err := deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		logger.Error.Printf("Failed to create deployment %s.\n", err)
	}
	logger.Info.Printf("Created deployment for plugin %q\n", result.GetObjectMeta().GetName())
	return err
}

func (rm *ResourceManager) ListJobs() (*batchv1.JobList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	list, err := rm.Clientset.BatchV1().Jobs(rm.Namespace).List(ctx, metav1.ListOptions{})
	return list, err
}

func (rm *ResourceManager) ListDeployments() (*appsv1.DeploymentList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	list, err := rm.Clientset.AppsV1().Deployments(rm.Namespace).List(ctx, metav1.ListOptions{})
	return list, err
}

// TerminateDeployment terminates given Kubernetes deployment
func (rm *ResourceManager) TerminateDeployment(pluginName string) error {
	pluginNameInLowcase := strings.ToLower(pluginName)

	deploymentsClient := rm.Clientset.AppsV1().Deployments(rm.Namespace)
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

func (rm *ResourceManager) TerminateJob(jobName string) error {
	return rm.Clientset.BatchV1().Jobs(rm.Namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{})
}

func (rm *ResourceManager) GetPluginStatus(jobName string) (string, error) {
	// TODO: Later we use pod name as we run plugins in one-shot?
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	job, err := rm.Clientset.BatchV1().Jobs(rm.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	selector := job.Spec.Selector
	labels, err := metav1.LabelSelectorAsSelector(selector)
	pods, err := rm.Clientset.CoreV1().Pods(rm.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.String()})
	if err != nil {
		return "", err
	}
	return string(pods.Items[0].Status.Phase), nil
}

func (rm *ResourceManager) GetPluginLog(jobName string, follow bool) (io.ReadCloser, error) {
	// TODO: Later we use pod name as we run plugins in one-shot?
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	job, err := rm.Clientset.BatchV1().Jobs(rm.Namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	selector := job.Spec.Selector
	labels, err := metav1.LabelSelectorAsSelector(selector)
	pods, err := rm.Clientset.CoreV1().Pods(rm.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.String()})
	if err != nil {
		return nil, err
	}
	switch pods.Items[0].Status.Phase {
	case apiv1.PodPending:
		return nil, fmt.Errorf("The plugin is in pending state")
	case apiv1.PodRunning:
		fallthrough
	case apiv1.PodSucceeded:
		fallthrough
	case apiv1.PodFailed:
		req := rm.Clientset.CoreV1().Pods(rm.Namespace).GetLogs(pods.Items[0].Name, &apiv1.PodLogOptions{Follow: follow})
		return req.Stream(context.TODO())
	}
	return nil, fmt.Errorf("The plugin (pod) is in %q state", string(pods.Items[0].Status.Phase))
	// podWatcher, err = rm.Clientset.CoreV1().Pods(rm.Namespace).Watch(ctx, metav1.ListOptions{LabelSelector: selector.String()})
}

func (rm *ResourceManager) Run(chanPluginToUpdate <-chan *datatype.Plugin) {
	for {
		plugin := <-chanPluginToUpdate
		logger.Debug.Printf("Plugin %s:%s needs to be in %s state", plugin.Name, plugin.Version, plugin.Status.SchedulingStatus)
		if plugin.Status.SchedulingStatus == datatype.Running {
			credential, err := rm.CreatePluginCredential(plugin)
			if err != nil {
				logger.Error.Printf("Could not create a plugin credential for %s on RabbitMQ at %s: %s", plugin.Name, rm.RMQManagement.Client.Endpoint, err.Error())
				continue
			}
			err = rm.RMQManagement.RegisterPluginCredential(credential)
			if err != nil {
				logger.Error.Printf("Could not register the credential %s to RabbitMQ at %s: %s", credential.Username, rm.RMQManagement.Client.Endpoint, err.Error())
				continue
			}
			deployablePlugin, err := rm.CreateDeployment(plugin, credential)
			if err != nil {
				logger.Error.Printf("Could not create a k3s deployment for plugin %s: %s", plugin.Name, err.Error())
				continue
			}
			err = rm.LaunchPlugin(deployablePlugin)
			if err != nil {
				logger.Error.Printf("Failed to launch plugin %s: %s", plugin.Name, err.Error())
			}
		} else if plugin.Status.SchedulingStatus == datatype.Stopped {
			err := rm.TerminateJob(plugin.Name)
			if err != nil {
				logger.Error.Printf("Failed to stop plugin %s: %s", plugin.Name, err.Error())
			}
		}
	}
}

// RMQManagement structs a connection to RMQManagement
type RMQManagement struct {
	RabbitmqManagementURI      string
	RabbitmqManagementUsername string
	RabbitmqManagementPassword string
	Client                     *rabbithole.Client
	Simulate                   bool
}

// NewRMQManagement creates and returns an instance of connection to RMQManagement
func NewRMQManagement(rmqManagementURI string, rmqManagementUsername string, rmqManagementPassword string, simulate bool) (*RMQManagement, error) {
	if simulate {
		return &RMQManagement{
			RabbitmqManagementURI:      rmqManagementURI,
			RabbitmqManagementUsername: rmqManagementUsername,
			RabbitmqManagementPassword: rmqManagementPassword,
			Client:                     nil,
			Simulate:                   simulate,
		}, nil
	}
	c, err := rabbithole.NewClient(rmqManagementURI, rmqManagementUsername, rmqManagementPassword)
	if err != nil {
		return nil, err
	}
	return &RMQManagement{
		RabbitmqManagementURI:      rmqManagementURI,
		RabbitmqManagementUsername: rmqManagementUsername,
		RabbitmqManagementPassword: rmqManagementPassword,
		Client:                     c,
		Simulate:                   simulate,
	}, nil
}

// RegisterPluginCredential registers given plugin credential to designated RMQ server
func (rmq *RMQManagement) RegisterPluginCredential(credential datatype.PluginCredential) error {
	// The functions below come from Sean's RunPlugin
	if _, err := rmq.Client.PutUser(credential.Username, rabbithole.UserSettings{
		Password: credential.Password,
	}); err != nil {
		return err
	}

	if _, err := rmq.Client.UpdatePermissionsIn("/", credential.Username, rabbithole.Permissions{
		Configure: "^amq.gen",
		Read:      ".*",
		Write:     ".*",
	}); err != nil {
		return err
	}
	logger.Debug.Printf("Plugin credential %s:%s is registered in RabbitMQ at %s", credential.Username, credential.Password, rmq.RabbitmqManagementURI)
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

func DetectDefaultKubeconfig() string {
	if _, err := os.ReadFile(rancherKubeconfigPath); err == nil {
		return rancherKubeconfigPath
	}
	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config")
	}
	return ""
}
