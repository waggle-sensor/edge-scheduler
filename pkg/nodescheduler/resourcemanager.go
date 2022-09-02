package nodescheduler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"

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
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

const (
	namespace             = "ses"
	rancherKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
	configMapNameForGoals = "waggle-plugin-scheduler-goals"
)

var (
	hostPathDirectoryOrCreate       = apiv1.HostPathDirectoryOrCreate
	hostPathDirectory               = apiv1.HostPathDirectory
	backOffLimit              int32 = 0
	ttlSecondsAfterFinished   int32 = 3600
)

// ResourceManager structs a resource manager talking to a local computing cluster to schedule plugins
type ResourceManager struct {
	Namespace     string
	Clientset     *kubernetes.Clientset
	MetricsClient *metrics.Clientset
	RMQManagement *RMQManagement
	Notifier      *interfacing.Notifier
	Simulate      bool
	Plugins       []*datatype.Plugin
	reserved      bool
	mutex         sync.Mutex
	runner        string
}

// NewResourceManager returns an instance of ResourceManager
func NewK3SResourceManager(incluster bool, kubeconfig string, runner string, simulate bool) (rm *ResourceManager, err error) {
	if simulate {
		return &ResourceManager{
			Namespace:     namespace,
			Clientset:     nil,
			MetricsClient: nil,
			Simulate:      simulate,
			Plugins:       make([]*datatype.Plugin, 0),
			reserved:      false,
			runner:        runner,
		}, nil
	}
	k3sClient, err := GetK3SClient(incluster, kubeconfig)
	if err != nil {
		return
	}
	metricsClient, err := GetK3SMetricsClient(incluster, kubeconfig)
	if err != nil {
		return
	}
	return &ResourceManager{
		Namespace:     namespace,
		Clientset:     k3sClient,
		MetricsClient: metricsClient,
		Simulate:      simulate,
		runner:        runner,
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

func (rm *ResourceManager) labelsForPlugin(plugin *datatype.Plugin) map[string]string {
	labels := map[string]string{
		"app":                           plugin.Name,
		"app.kubernetes.io/name":        plugin.Name,
		"app.kubernetes.io/managed-by":  rm.runner,
		"app.kubernetes.io/created-by":  rm.runner,
		"sagecontinuum.org/plugin-job":  plugin.PluginSpec.Job,
		"sagecontinuum.org/plugin-task": plugin.Name,
	}

	// in develop mode, we omit the role labels to opt out of network traffic filtering
	// this is intended to do things like:
	// * allow developers to initially pull from github and add packages
	// * allow interfacing with devices in wan subnet until we add site specific exceptions
	if !plugin.PluginSpec.DevelopMode {
		labels["role"] = "plugin"
		labels["sagecontinuum.org/role"] = "plugin"
	}

	return labels
}

func nodeSelectorForConfig(pluginSpec *datatype.PluginSpec) map[string]string {
	vals := map[string]string{}
	if pluginSpec.Node != "" {
		vals["k3s.io/hostname"] = pluginSpec.Node
	}
	for k, v := range pluginSpec.Selector {
		vals[k] = v
	}
	return vals
}

func securityContextForConfig(pluginSpec *datatype.PluginSpec) *apiv1.SecurityContext {
	if pluginSpec.Privileged {
		return &apiv1.SecurityContext{Privileged: &pluginSpec.Privileged}
	}
	return nil
}

func (rm *ResourceManager) ConfigureKubernetes(inCluster bool, kubeconfig string) error {
	k3sClient, err := GetK3SClient(inCluster, kubeconfig)
	if err != nil {
		return err
	}
	rm.Clientset = k3sClient
	metricsClient, err := GetK3SMetricsClient(inCluster, kubeconfig)
	if err != nil {
		return err
	}
	rm.MetricsClient = metricsClient
	return nil
}

// CreatePluginCredential creates a credential inside RabbitMQ server for the plugin
func (rm *ResourceManager) CreatePluginCredential(plugin *datatype.Plugin) (datatype.PluginCredential, error) {
	tag, err := plugin.PluginSpec.GetImageTag()
	if err != nil {
		return datatype.PluginCredential{}, err
	}

	// TODO: We will need to add instance of plugin as a aprt of Username
	// username should follow "plugin.NAME:VERSION" format to publish messages via WES
	credential := datatype.PluginCredential{
		Username: fmt.Sprint("plugin.", strings.ToLower(plugin.Name), ":", tag),
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

// CopyConfigMap copies Kubernetes ConfigMap between namespaces
func (rm *ResourceManager) CopyConfigMap(configMapName string, fromNamespace string, toNamespace string) error {
	configMap, err := rm.Clientset.CoreV1().ConfigMaps(fromNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return rm.CreateConfigMap(configMap.Name, configMap.Data, toNamespace, true)
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

func (rm *ResourceManager) CreateConfigMap(name string, data map[string]string, namespace string, overwrite bool) error {
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
	if overwrite {
		_, err = rm.Clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), &config, metav1.UpdateOptions{})
	}
	return err
}

// WatchConfigMap
func (rm *ResourceManager) GetConfigMapWatcher(name string, namespace string) (func() (watch.Interface, error), error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	configMap, err := rm.Clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// var selector *metav1.LabelSelector
	// err = metav1.Convert_Map_string_To_string_To_v1_LabelSelector(&configMap.Labels, selector, nil)
	return func() (watch.Interface, error) {
		return rm.Clientset.CoreV1().
			ConfigMaps(namespace).
			Watch(context.TODO(),
				metav1.SingleObject(metav1.ObjectMeta{Name: configMap.Name, Namespace: configMap.Namespace}),
			)
	}, nil
}

func (rm *ResourceManager) WatchJob(name string, namespace string, retry int) (watcher watch.Interface, err error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	for i := 0; i <= retry; i++ {
		job, err := rm.Clientset.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			logger.Debug.Printf("Failed to job %q", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		// var selector *metav1.LabelSelector
		// err = metav1.Convert_Map_string_To_string_To_v1_LabelSelector(&configMap.Labels, selector, nil)
		watcher, err = rm.Clientset.BatchV1().Jobs(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: job.Name, Namespace: job.Namespace}))
		if err != nil {
			logger.Debug.Printf("Failed to get watcher for %q: %q", job.Name, err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
	return
}

func (rm *ResourceManager) WatchJobs(namespace string) (watch.Interface, error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	watcher, err := rm.Clientset.BatchV1().Jobs(namespace).Watch(context.TODO(), metav1.ListOptions{})
	return watcher, err
}

func (rm *ResourceManager) createPodTemplateSpecForPlugin(plugin *datatype.Plugin) (v1.PodTemplateSpec, error) {
	envs := []apiv1.EnvVar{
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
		// Use WES scoreboard
		{
			Name:  "REDIS_HOST",
			Value: "wes-scoreboard.default.svc.cluster.local",
		},
	}

	for k, v := range plugin.PluginSpec.Env {
		envs = append(envs, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	tag, err := plugin.PluginSpec.GetImageTag()
	if err != nil {
		return v1.PodTemplateSpec{}, err
	}

	volumes := []apiv1.Volume{
		{
			Name: "uploads",
			VolumeSource: apiv1.VolumeSource{
				HostPath: &apiv1.HostPathVolumeSource{
					Path: path.Join("/media/plugin-data/uploads", plugin.PluginSpec.Job, plugin.Name, tag),
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
	}

	volumeMounts := []apiv1.VolumeMount{
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
	}

	// provide privileged plugins access to host devices
	if plugin.PluginSpec.Privileged {
		volumes = append(volumes, apiv1.Volume{
			Name: "dev",
			VolumeSource: apiv1.VolumeSource{
				HostPath: &apiv1.HostPathVolumeSource{
					Path: "/dev",
					Type: &hostPathDirectory,
				},
			},
		})
		volumeMounts = append(volumeMounts, apiv1.VolumeMount{
			Name:      "dev",
			MountPath: "/host/dev",
		})
	}

	appMeta := struct {
		Host   string `json:"host"`
		Job    string `json:"job"`
		Task   string `json:"task"`
		Plugin string `json:"plugin"`
	}{
		Host:   "$(HOST)",
		Task:   plugin.Name,
		Job:    plugin.PluginSpec.Job,
		Plugin: plugin.PluginSpec.Image,
	}

	appMetaData, err := json.Marshal(appMeta)
	if err != nil {
		// since we control the contents, this should never fail
		panic(err)
	}

	initContainers := []apiv1.Container{
		{
			Name:  "init-app-meta-cache",
			Image: "redis:7.0.4",
			Command: []string{
				"redis-cli",
				"-h",
				"wes-app-meta-cache",
				"SET",
				"app-meta.$(WAGGLE_APP_ID)",
				string(appMetaData),
			},
			Env: []apiv1.EnvVar{
				{
					Name: "WAGGLE_APP_ID",
					ValueFrom: &apiv1.EnvVarSource{
						FieldRef: &apiv1.ObjectFieldSelector{
							FieldPath: "metadata.uid",
						},
					},
				},
				{
					Name: "HOST",
					ValueFrom: &apiv1.EnvVarSource{
						FieldRef: &apiv1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
		},
	}

	containers := []apiv1.Container{
		{
			SecurityContext: securityContextForConfig(plugin.PluginSpec),
			Name:            plugin.Name,
			Image:           plugin.PluginSpec.Image,
			Args:            plugin.PluginSpec.Args,
			Env:             envs,
			Resources: apiv1.ResourceRequirements{
				Limits:   apiv1.ResourceList{},
				Requests: apiv1.ResourceList{},
				// TODO: this should be revisited when talking about resource-aware scheduling
				// Requests: apiv1.ResourceList{
				// 	apiv1.ResourceCPU:    resource.MustParse("1500m"),
				// 	apiv1.ResourceMemory: resource.MustParse("1.5Gi"),
				// },
			},
			VolumeMounts: volumeMounts,
		},
	}

	if plugin.PluginSpec.Entrypoint != "" {
		containers[0].Command = []string{plugin.PluginSpec.Entrypoint}
	}

	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: rm.labelsForPlugin(plugin),
		},
		Spec: apiv1.PodSpec{
			PriorityClassName: "wes-app-priority",
			NodeSelector:      nodeSelectorForConfig(plugin.PluginSpec),
			// TODO: The priority class will be revisited when using resource metrics to schedule plugins
			InitContainers: initContainers,
			Containers:     containers,
			Volumes:        volumes,
		},
	}, nil
}

// CreateK3SJob creates and returns a Kubernetes job object of the pllugin
func (rm *ResourceManager) CreateJob(plugin *datatype.Plugin) (*batchv1.Job, error) {
	name, err := pluginNameForSpecDeployment(plugin)
	if err != nil {
		return nil, err
	}

	template, err := rm.createPodTemplateSpecForPlugin(plugin)
	if err != nil {
		return nil, err
	}
	template.Spec.RestartPolicy = apiv1.RestartPolicyNever

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rm.Namespace,
			Labels:    template.Labels,
		},
		Spec: batchv1.JobSpec{
			Template:                template,
			BackoffLimit:            &backOffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
		},
	}, nil
}

// CreateDeployment creates and returns a Kubernetes deployment object of the plugin
// It also embeds a K3S configmap for plugin if needed
func (rm *ResourceManager) CreateDeployment(plugin *datatype.Plugin) (*appsv1.Deployment, error) {
	name, err := pluginNameForSpecDeployment(plugin)
	if err != nil {
		return nil, err
	}

	template, err := rm.createPodTemplateSpecForPlugin(plugin)
	if err != nil {
		return nil, err
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rm.Namespace,
			Labels:    template.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: template.Labels,
			},
			Template: template,
		},
	}, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	jobClient := rm.Clientset.BatchV1().Jobs(rm.Namespace)
	return jobClient.Create(ctx, job, metav1.CreateOptions{})
}

// LaunchPlugin launches a k3s deployment in the cluster
func (rm *ResourceManager) LaunchPlugin(deployment *appsv1.Deployment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	deleteDependencies := metav1.DeletePropagationForeground
	return rm.Clientset.BatchV1().Jobs(rm.Namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{PropagationPolicy: &deleteDependencies})
}

func (rm *ResourceManager) GetPluginStatus(jobName string) (apiv1.PodPhase, error) {
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
	if len(pods.Items) > 0 {
		return pods.Items[0].Status.Phase, nil
	} else {
		return "", fmt.Errorf("No pod exists for job %q", jobName)
	}
}

func (rm *ResourceManager) GetPod(jobName string) (*apiv1.Pod, error) {
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
	if len(pods.Items) > 0 {
		return &pods.Items[0], nil
	} else {
		return nil, fmt.Errorf("No pod exists for job %q", jobName)
	}
}

func (rm *ResourceManager) GetPodName(jobName string) (string, error) {
	pod, err := rm.GetPod(jobName)
	if err != nil {
		return "", err
	}
	return pod.Name, nil
}

func (rm *ResourceManager) GetPluginLog(jobName string, follow bool) (io.ReadCloser, error) {
	pod, err := rm.GetPod(jobName)
	if err != nil {
		return nil, err
	}
	switch pod.Status.Phase {
	case apiv1.PodPending:
		return nil, fmt.Errorf("The plugin is in pending state")
	case apiv1.PodRunning:
		fallthrough
	case apiv1.PodSucceeded:
		fallthrough
	case apiv1.PodFailed:
		req := rm.Clientset.CoreV1().Pods(rm.Namespace).GetLogs(pod.Name, &apiv1.PodLogOptions{Follow: follow})
		return req.Stream(context.TODO())
	}
	return nil, fmt.Errorf("The plugin (pod) is in %q state", string(pod.Status.Phase))
	// podWatcher, err = rm.Clientset.CoreV1().Pods(rm.Namespace).Watch(ctx, metav1.ListOptions{LabelSelector: selector.String()})
}

// CleanUp removes all currently running jobs
func (rm *ResourceManager) CleanUp() error {
	jobs, err := rm.ListJobs()
	if err != nil {
		return err
	}
	for _, job := range jobs.Items {
		// Skip WES service jobs
		if strings.Contains(job.Name, "wes") {
			continue
		}
		podStatus, err := rm.GetPluginStatus(job.Name)
		if err != nil {
			logger.Debug.Printf("Failed to read %q's pod status. Skipping...", job.Name)
			continue
		}
		logger.Debug.Printf("Job %q's pod status %q", job.Name, podStatus)
		switch podStatus {
		case apiv1.PodPending, apiv1.PodRunning:
			rm.TerminateJob(job.Name)
			logger.Debug.Printf("Job %q terminated successfully", job.Name)
		}
	}
	return nil
}

func (rm *ResourceManager) UpdateReservation(value bool) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.reserved = value
}

func (rm *ResourceManager) WillItFit(plugin *datatype.Plugin) bool {
	if rm.reserved {
		return false
	} else {
		return true
	}
}

func (rm *ResourceManager) LaunchAndWatchPlugin(plugin *datatype.Plugin) {
	logger.Debug.Printf("Running plugin %q...", plugin.Name)
	job, err := rm.CreateJob(plugin)
	if err != nil {
		logger.Error.Printf("Failed to create Kubernetes Job for %q: %q", plugin.Name, err.Error())
		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(err.Error()).AddPluginMeta(plugin).Build())
		return
	}
	_, err = rm.RunPlugin(job)
	defer rm.TerminateJob(job.Name)
	if err != nil {
		logger.Error.Printf("Failed to run %q: %q", job.Name, err.Error())
		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(err.Error()).AddPluginMeta(plugin).Build())
		return
	}
	logger.Info.Printf("Plugin %q deployed", job.Name)
	plugin.PluginSpec.Job = job.Name
	// rm.UpdateReservation(true)
	watcher, err := rm.WatchJob(job.Name, rm.Namespace, 3)
	if err != nil {
		logger.Error.Printf("Failed to watch %q. Abort the execution", job.Name)
		rm.TerminateJob(job.Name)
		// rm.UpdateReservation(false)
		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(err.Error()).AddK3SJobMeta(job).AddPluginMeta(plugin).Build())
		return
	}
	chanEvent := watcher.ResultChan()
	defer watcher.Stop()
	for {
		event := <-chanEvent
		job := event.Object.(*batchv1.Job)
		switch event.Type {
		case watch.Added:
			pod, _ := rm.GetPod(job.Name)
			rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusLaunched).AddK3SJobMeta(job).AddPodMeta(pod).AddPluginMeta(plugin).Build())
		case watch.Modified:
			if len(job.Status.Conditions) > 0 {
				pod, _ := rm.GetPod(job.Name)
				logger.Debug.Printf("Plugin %s status %s: %s", job.Name, event.Type, job.Status.Conditions[0].Type)
				switch job.Status.Conditions[0].Type {
				case batchv1.JobComplete:
					// rm.UpdateReservation(false)
					rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusComplete).AddK3SJobMeta(job).AddPodMeta(pod).AddPluginMeta(plugin).Build())
					return
				case batchv1.JobFailed:
					// rm.UpdateReservation(false)
					rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(job.Status.Conditions[0].Reason).AddK3SJobMeta(job).AddPodMeta(pod).AddPluginMeta(plugin).Build())
					return
				}
			} else {
				logger.Debug.Printf("Plugin %s status %s: %s", job.Name, event.Type, "UNKNOWN")
			}
		case watch.Deleted:
			logger.Debug.Printf("Plugin got deleted. Returning resource and notify")
			// rm.UpdateReservation(false)
			rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason("Plugin deleted").AddK3SJobMeta(job).AddPluginMeta(plugin).Build())
			return
		case watch.Error:
			logger.Debug.Printf("Error on watcher. Returning resource and notify")
			rm.TerminateJob(job.Name)
			// rm.UpdateReservation(false)
			rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason("Error on watcher").AddK3SJobMeta(job).AddPluginMeta(plugin).Build())
			return
		}
	}
}

func (rm *ResourceManager) GatherResourceUse() {
	// rm.Clientset.metricsv1
}

// RunGabageCollector cleans up completed/failed jobs that exceed
// their lifespan specified in `ttlSecondsAfterFinished`
//
// NOTE: This should be done by Kubernetes TTL controller with TTLSecondsAfterFinished specified in job,
// but it could not get enabled in k3s with Kubernetes v1.20 that has the TTL controller disabled
// by default. Kubernetes v1.21+ has the controller enabled by default
func (rm *ResourceManager) RunGabageCollector() error {
	jobList, err := rm.ListJobs()
	if err != nil {
		return fmt.Errorf("Failed to get job list: %s", err.Error())
	}
	for _, job := range jobList.Items {
		podPhase, err := rm.GetPluginStatus(job.Name)
		if err != nil {
			continue
		}
		switch podPhase {
		case apiv1.PodFailed:
			fallthrough
		case apiv1.PodSucceeded:
			elapsedSeconds := time.Now().Sub(job.CreationTimestamp.Time).Seconds()
			if elapsedSeconds > float64(ttlSecondsAfterFinished) {
				logger.Debug.Printf("%q exceeded ttlSeconds of %.2f. Cleaning up...", job.Name, elapsedSeconds)
				rm.TerminateJob(job.Name)
			}
		}
	}
	return nil
}

func (rm *ResourceManager) Configure() (err error) {
	err = rm.CreateNamespace("ses")
	if err != nil {
		return
	}
	servicesToBringUp := []string{"wes-rabbitmq", "wes-audio-server", "wes-scoreboard", "wes-app-meta-cache"}
	for _, service := range servicesToBringUp {
		err = rm.ForwardService(service, "default", "ses")
		if err != nil {
			return
		}
	}
	configMapsToBring := []string{"waggle-data-config", "wes-audio-server-plugin-conf"}
	for _, configMapName := range configMapsToBring {
		err = rm.CopyConfigMap(configMapName, "default", rm.Namespace)
		if err != nil {
			logger.Error.Printf("Failed to create ConfigMap %q: %q", configMapName, err.Error())
		}
	}
	err = rm.CreateConfigMap(configMapNameForGoals, map[string]string{}, "default", false)
	if err != nil {
		return
	}
	return
}

func (rm *ResourceManager) Run(chanPluginToUpdate <-chan *datatype.Plugin) {
	if rm.MetricsClient == nil {
		logger.Info.Println("No metrics client is set. Metrics information cannot be obtained")
	}
	// NOTE: The garbage collector runs to clean up completed/failed jobs
	//       This should be done by Kubernetes with versions higher than v1.21
	//       v1.20 could do it by enabling TTL controller, but could not set it
	//       via k3s server --kube-control-manager-arg feature-gates=TTL...=true
	// go ns.ResourceManager.RunGabageCollector()
	gabageCollectorTicker := time.NewTicker(1 * time.Minute)
	logger.Info.Printf("Pull goals from k3s configmap %s", configMapNameForGoals)
	goalConfigMapFunc, _ := rm.GetConfigMapWatcher(configMapNameForGoals, "default")
	goalWatcher := NewAdvancedWatcher(configMapNameForGoals, goalConfigMapFunc)
	goalWatcher.Run()
	logger.Info.Println("Starting the main loop of resource manager...")
	for {
		select {
		case <-gabageCollectorTicker.C:
			err := rm.RunGabageCollector()
			if err != nil {
				logger.Error.Printf("Failed to run gabage collector: %s", err.Error())
			}
		case event := <-goalWatcher.C:
			switch event.Type {
			case watch.Added, watch.Modified:
				if updatedConfigMap, ok := event.Object.(*apiv1.ConfigMap); ok {
					logger.Debug.Printf("%v", updatedConfigMap.Data)
					event := datatype.NewEventBuilder(datatype.EventGoalStatusReceivedBulk).
						AddEntry("goals", updatedConfigMap.Data["goals"]).Build()
					rm.Notifier.Notify(event)
				}
			}
			// case watch.Deleted, watch.Error:
			// 	logger.Error.Printf("Failed on %q k3s watcher", configMapName)
			// 	break
			// }
		}
	}
	// for {
	// 	nodeMetrics, err := rm.MetricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	// 	if err != nil {
	// 		logger.Error.Println("Error:", err)
	// 		return
	// 	}
	// 	for _, nodeMetric := range nodeMetrics.Items {
	// 		cpuQuantity, ok := nodeMetric.Usage.Cpu().AsInt64()
	// 		memQuantity, ok := nodeMetric.Usage.Memory().AsInt64()
	// 		if !ok {
	// 			return
	// 		}
	// 		msg := fmt.Sprintf("Node Name: %s \n CPU usage: %d \n Memory usage: %d", nodeMetric.Name, cpuQuantity, memQuantity)
	// 		logger.Debug.Println(msg)
	// 	}
	// }
	// podMetrics, err := rm.MetricsClient.MetricsV1beta1().PodMetricses(rm.Namespace).List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	logger.Error.Println("Error:", err)
	// 	return
	// }
	// for _, podMetric := range podMetrics.Items {
	// 	podContainers := podMetric.Containers
	// 	for _, container := range podContainers {
	// 		cpuQuantity, ok := container.Usage.Cpu().AsInt64()
	// 		memQuantity, ok := container.Usage.Memory().AsInt64()
	// 		if !ok {
	// 			return
	// 		}
	// 		msg := fmt.Sprintf("Container Name: %s \n CPU usage: %d \n Memory usage: %d", container.Name, cpuQuantity, memQuantity)
	// 		logger.Debug.Println(msg)
	// 	}
	// }
	// 	time.Sleep(1 * time.Millisecond)
	// }

	// NOTE: Kubernetes did not allow to watch for metrics
	// ERROR: 2022/01/05 13:55:48 resourcemanager.go:873: Error: the server does not allow this method on the requested resource (get pods.metrics.k8s.io)
	// watcher, err := rm.MetricsClient.MetricsV1beta1().PodMetricses(rm.Namespace).Watch(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	logger.Error.Println("Error:", err)
	// 	return
	// }
	// chanEvent := watcher.ResultChan()
	// for {
	// 	event := <-chanEvent
	// 	switch event.Type {
	// 	case watch.Added:
	// 		fallthrough
	// 	case watch.Modified:
	// 		podMetrics := event.Object.(*v1beta1.PodMetrics)
	// 		for _, container := range podMetrics.Containers {
	// 			cpuQuantity, ok := container.Usage.Cpu().AsInt64()
	// 			memQuantity, ok := container.Usage.Memory().AsInt64()
	// 			if !ok {
	// 				return
	// 			}
	// 			msg := fmt.Sprintf("Container Name: %s \n CPU usage: %d \n Memory usage: %d", container.Name, cpuQuantity, memQuantity)
	// 			logger.Debug.Println(msg)
	// 		}
	// 	}
	// }

	// for {
	// 	select {
	// 	case plugin := <-chanPluginToUpdate:
	// 		logger.Debug.Printf("Plugin status changed to %q", plugin.Status.SchedulingStatus)
	// 		switch plugin.Status.SchedulingStatus {
	// 		case datatype.Ready:
	// 			logger.Debug.Printf("Running the plugin %q...", plugin.Name)
	// 			job, err := rm.CreateJob(plugin)
	// 			if err != nil {
	// 				logger.Error.Printf("Failed to create Kubernetes Job for %q: %q", plugin.Name, err.Error())
	// 			} else {
	// 				_, err = rm.RunPlugin(job)
	// 				if err != nil {
	// 					logger.Error.Printf("Failed to run %q: %q", plugin.Name, err.Error())
	// 				} else {
	// 					logger.Info.Printf("Plugin %q deployed", plugin.Name)
	// 					plugin.UpdatePluginSchedulingStatus(datatype.Running)
	// 					rm.UpdateReservation(true)
	// 					go func() {
	// 						watcher, err := rm.WatchJob(plugin.Name, rm.Namespace, 0)
	// 						if err != nil {
	// 							logger.Error.Printf("Failed to watch %q. Abort the execution", plugin.Name)
	// 							rm.TerminateJob(job.Name)
	// 						}
	// 						chanEvent := watcher.ResultChan()
	// 						for {
	// 							event := <-chanEvent
	// 							switch event.Type {
	// 							case watch.Added:
	// 								fallthrough
	// 							case watch.Deleted:
	// 								fallthrough
	// 							case watch.Modified:
	// 								job := event.Object.(*batchv1.Job)
	// 								if len(job.Status.Conditions) > 0 {
	// 									logger.Debug.Printf("%s: %s", event.Type, job.Status.Conditions[0].Type)
	// 									switch job.Status.Conditions[0].Type {
	// 									case batchv1.JobComplete:
	// 										fallthrough
	// 									case batchv1.JobFailed:
	// 										plugin.UpdatePluginSchedulingStatus(datatype.Waiting)
	// 										rm.UpdateReservation(false)
	// 									}
	// 								} else {
	// 									logger.Debug.Printf("%s: %s", event.Type, "UNKNOWN")
	// 								}
	// 							}
	// 						}
	// 					}()
	// 				}
	// 			}
	// 		}
	// 	}
	// if plugin.Status.SchedulingStatus == datatype.Running {
	// 	credential, err := rm.CreatePluginCredential(plugin)
	// 	if err != nil {
	// 		logger.Error.Printf("Could not create a plugin credential for %s on RabbitMQ at %s: %s", plugin.Name, rm.RMQManagement.Client.Endpoint, err.Error())
	// 		continue
	// 	}
	// 	err = rm.RMQManagement.RegisterPluginCredential(credential)
	// 	if err != nil {
	// 		logger.Error.Printf("Could not register the credential %s to RabbitMQ at %s: %s", credential.Username, rm.RMQManagement.Client.Endpoint, err.Error())
	// 		continue
	// 	}
	// 	deployablePlugin, err := rm.CreateDeployment(plugin, credential)
	// 	if err != nil {
	// 		logger.Error.Printf("Could not create a k3s deployment for plugin %s: %s", plugin.Name, err.Error())
	// 		continue
	// 	}
	// 	err = rm.LaunchPlugin(deployablePlugin)
	// 	if err != nil {
	// 		logger.Error.Printf("Failed to launch plugin %s: %s", plugin.Name, err.Error())
	// 	}
	// } else if plugin.Status.SchedulingStatus == datatype.Stopped {
	// 	err := rm.TerminateJob(plugin.Name)
	// 	if err != nil {
	// 		logger.Error.Printf("Failed to stop plugin %s: %s", plugin.Name, err.Error())
	// 	}
	// }
	// }
}

type AdvancedWatcher struct {
	Name string
	Func func() (watch.Interface, error)
	C    chan watch.Event
}

func NewAdvancedWatcher(n string, f func() (watch.Interface, error)) *AdvancedWatcher {
	return &AdvancedWatcher{
		Name: n,
		Func: f,
		C:    make(chan watch.Event),
	}
}

func (w *AdvancedWatcher) runWatcher() error {
	ticker := time.NewTicker(1 * time.Minute)
	lastEvent := time.Now()
	timeOut := 30 * time.Minute
	watcher, err := w.Func()
	if err != nil {
		return fmt.Errorf("Failed to get watcher from Kubernetes")
	}
	for {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("Watcher is closed")
			} else {
				w.C <- e
				lastEvent = time.Now()
			}
		case <-ticker.C:
			if time.Now().After(lastEvent.Add(timeOut)) {
				return fmt.Errorf("Watcher timed out")
			}
		}
	}
}

func (w *AdvancedWatcher) Run() {
	go func() {
		for {
			if err := w.runWatcher(); err != nil {
				logger.Error.Printf("Failed on watcher %q: %s", w.Name, err.Error())
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

var validNamePattern = regexp.MustCompile("^[a-z0-9-]+$")

func pluginNameForSpecJob(plugin *datatype.Plugin) (string, error) {
	// if no given name for the plugin, use PLUGIN-VERSION-INSTANCE format for name
	// INSTANCE is calculated as Sha256("DOMAIN/PLUGIN:VERSION&ARGUMENTS") and
	// take the first 8 hex letters.
	// NOTE: if multiple plugins with the same version and arguments are given for
	//       the same domain, only one deployment will be applied to the cluster
	// NOTE2: To comply with RFC 1123 for Kubernetes object name, only lower alphanumeric
	//        characters with '-' is allowed
	if plugin.Name != "" {
		jobName := strings.Join([]string{plugin.Name, strconv.FormatInt(time.Now().Unix(), 10)}, "-")
		if !validNamePattern.MatchString(jobName) {
			return "", fmt.Errorf("plugin name must consist of alphanumeric characters with '-' RFC1123")
		}
		return jobName, nil
	}
	return generateJobNameForSpec(plugin.PluginSpec)
}

// TODO(sean) consolidate with other name setup
func pluginNameForSpecDeployment(plugin *datatype.Plugin) (string, error) {
	// if no given name for the plugin, use PLUGIN-VERSION-INSTANCE format for name
	// INSTANCE is calculated as Sha256("DOMAIN/PLUGIN:VERSION&ARGUMENTS") and
	// take the first 8 hex letters.
	// NOTE: if multiple plugins with the same version and arguments are given for
	//       the same domain, only one deployment will be applied to the cluster
	// NOTE2: To comply with RFC 1123 for Kubernetes object name, only lower alphanumeric
	//        characters with '-' is allowed
	if plugin.Name != "" {
		if !validNamePattern.MatchString(plugin.Name) {
			return "", fmt.Errorf("plugin name must consist of alphanumeric characters with '-' RFC1123")
		}
		return plugin.Name, nil
	}
	return generateJobNameForSpec(plugin.PluginSpec)
}

// generateJobNameForSpec generates a consistent name for a Spec.
//
// Very important note from: https://pkg.go.dev/encoding/json#Marshal
//
// Map values encode as JSON objects. The map's key type must either be a string, an integer type,
// or implement encoding.TextMarshaler. The map keys are sorted and used as JSON object keys by applying
// the following rules, subject to the UTF-8 coercion described for string values above:
//
// The "map keys are sorted" bit is important for us as it allows us to ensure the hash is consistent.
func generateJobNameForSpec(spec *datatype.PluginSpec) (string, error) {
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

// GetK3SMetricsClient returns an instance of Metrics clientset talking to a K3S cluster
func GetK3SMetricsClient(incluster bool, pathToConfig string) (*metrics.Clientset, error) {
	if incluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return metrics.NewForConfig(config)
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", pathToConfig)
		if err != nil {
			return nil, err
		}
		return metrics.NewForConfig(config)
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
