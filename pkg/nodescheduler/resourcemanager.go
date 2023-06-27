package nodescheduler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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
	"k8s.io/apimachinery/pkg/api/resource"
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
	letters               = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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

func generateRandomString(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func (rm *ResourceManager) labelsForPlugin(plugin *datatype.Plugin) map[string]string {
	labels := map[string]string{
		"app":                               plugin.Name,
		"app.kubernetes.io/name":            plugin.Name,
		"app.kubernetes.io/managed-by":      rm.runner,
		"app.kubernetes.io/created-by":      rm.runner,
		"sagecontinuum.org/plugin-job":      plugin.PluginSpec.Job,
		"sagecontinuum.org/plugin-task":     plugin.Name,
		"sagecontinuum.org/plugin-instance": plugin.Name + "-" + generateRandomString(6),
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

func resourceListForConfig(pluginSpec *datatype.PluginSpec) (apiv1.ResourceRequirements, error) {
	resources := apiv1.ResourceRequirements{
		Limits:   apiv1.ResourceList{},
		Requests: apiv1.ResourceList{},
	}
	for resourceName, quantityString := range pluginSpec.Resource {
		quantity, err := resource.ParseQuantity(quantityString)
		if err != nil {
			return resources, fmt.Errorf("Failed to parse %q: %s", quantityString, err.Error())
		}
		switch resourceName {
		case "limit.cpu":
			resources.Limits[apiv1.ResourceCPU] = quantity
		case "limit.memory":
			resources.Limits[apiv1.ResourceMemory] = quantity
		case "limit.gpu":
			resources.Limits["nvidia.com/gpu"] = quantity
		case "request.cpu":
			resources.Requests[apiv1.ResourceCPU] = quantity
		case "request.memory":
			resources.Requests[apiv1.ResourceMemory] = quantity
		default:
			return resources, fmt.Errorf("Unknown resource name %q", resourceName)
		}
	}
	return resources, nil
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
			logger.Debug.Printf("Failed to get job %q", err.Error())
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

func (rm *ResourceManager) WatchPod(name string, namespace string, retry int) (watcher watch.Interface, err error) {
	if namespace == "" {
		namespace = rm.Namespace
	}
	for i := 0; i <= retry; i++ {
		pod, err := rm.Clientset.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			logger.Debug.Printf("Failed to get pod %q", err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		// var selector *metav1.LabelSelector
		// err = metav1.Convert_Map_string_To_string_To_v1_LabelSelector(&configMap.Labels, selector, nil)
		watcher, err = rm.Clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace}))
		if err != nil {
			logger.Debug.Printf("Failed to get watcher for %q: %q", pod.Name, err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
	return
}

func (rm *ResourceManager) createPodTemplateSpecForPlugin(pr *datatype.PluginRuntime) (v1.PodTemplateSpec, error) {
	envs := []apiv1.EnvVar{
		{
			Name:  "PULSE_SERVER",
			Value: "tcp:wes-audio-server.default.svc.cluster.local:4713",
		},
		{
			Name:  "WAGGLE_PLUGIN_HOST",
			Value: "wes-rabbitmq.default.svc.cluster.local",
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
		{
			Name:  "WAGGLE_GPS_SERVER",
			Value: "wes-gps-server.default.svc.cluster.local",
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
			Name:  "WAGGLE_SCOREBOARD",
			Value: "wes-scoreboard.default.svc.cluster.local",
		},
		{
			Name: "HOST",
			ValueFrom: &apiv1.EnvVarSource{
				FieldRef: &apiv1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}

	for k, v := range pr.Plugin.PluginSpec.Env {
		envs = append(envs, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	tag, err := pr.Plugin.PluginSpec.GetImageTag()
	if err != nil {
		return v1.PodTemplateSpec{}, err
	}

	volumes := []apiv1.Volume{
		{
			Name: "uploads",
			VolumeSource: apiv1.VolumeSource{
				HostPath: &apiv1.HostPathVolumeSource{
					Path: path.Join("/media/plugin-data/uploads", pr.Plugin.PluginSpec.Job, pr.Plugin.Name, tag),
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

	if pr.EnablePluginController {
		volumes = append(volumes, apiv1.Volume{
			Name: "local-share",
			VolumeSource: apiv1.VolumeSource{
				EmptyDir: &apiv1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, apiv1.VolumeMount{
			Name:      "local-share",
			MountPath: "/waggle",
		})
	}

	// provide privileged plugins access to host devices
	if pr.Plugin.PluginSpec.Privileged {
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
		Task:   pr.Plugin.Name,
		Job:    pr.Plugin.PluginSpec.Job,
		Plugin: pr.Plugin.PluginSpec.Image,
	}

	appMetaData, err := json.Marshal(appMeta)
	if err != nil {
		// since we control the contents, this should never fail
		panic(err)
	}

	initContainers := []apiv1.Container{
		{
			Name:  "init-app-meta-cache",
			Image: "waggle/app-meta-cache:0.1.1",
			Command: []string{
				"/update-app-cache",
				"set",
				"--nodename",
				"$(HOST)",
				"--host",
				"wes-app-meta-cache.default.svc.cluster.local",
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

	resources, err := resourceListForConfig(pr.Plugin.PluginSpec)
	if err != nil {
		return v1.PodTemplateSpec{}, err
	}

	containers := []apiv1.Container{
		{
			SecurityContext: securityContextForConfig(pr.Plugin.PluginSpec),
			Name:            pr.Plugin.Name,
			Image:           pr.Plugin.PluginSpec.Image,
			Args:            pr.Plugin.PluginSpec.Args,
			Env:             envs,
			Resources:       resources,
			VolumeMounts:    volumeMounts,
		},
	}

	if pr.Plugin.PluginSpec.Entrypoint != "" {
		containers[0].Command = []string{pr.Plugin.PluginSpec.Entrypoint}
	}

	// add plugin-controller sidecar container
	if pr.EnablePluginController {
		logger.Info.Printf("plugin-controller sidecar is added to %s", pr.Plugin.Name)
		pluginControllerArgs := []string{
			"--enable-cpu-performance",
			"--enable-metrics-publishing",
		}
		if len(containers[0].Command) >= 1 {
			pluginProcessName := containers[0].Command[0]
			logger.Info.Printf("user specified plugin process (%s). it will be passed to the plugin-controller", pluginProcessName)
			pluginControllerArgs = append(pluginControllerArgs, "--plugin-process-name", pluginProcessName)
		}
		if _, found := pr.Plugin.PluginSpec.Selector["resource.gpu"]; found {
			logger.Info.Printf("%s's plugin-controller will collect GPU performance", pr.Plugin.Name)
			pluginControllerArgs = append(pluginControllerArgs, "--enable-gpu-performance")
		}
		// adding plugin-controller to the pod
		containers = append(containers, apiv1.Container{
			Name: "plugin-controller",
			// Image: "waggle/plugin-controller:0.2.0",
			Image: "10.31.81.1:5000/local/plugin-controller",
			Args:  pluginControllerArgs,
			Env: []apiv1.EnvVar{
				{
					Name:  "GPU_METRIC_HOST",
					Value: "wes-jetson-exporter.default.svc.cluster.local",
				},
				{
					Name:  "WAGGLE_PLUGIN_HOST",
					Value: "wes-rabbitmq.default.svc.cluster.local",
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
				{
					Name: "WAGGLE_APP_ID",
					ValueFrom: &apiv1.EnvVarSource{
						FieldRef: &apiv1.ObjectFieldSelector{
							FieldPath: "metadata.uid",
						},
					},
				},
			},
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "local-share",
					MountPath: "/app/",
					ReadOnly:  true,
				},
			},
			Resources: apiv1.ResourceRequirements{
				Limits: apiv1.ResourceList{},
				Requests: apiv1.ResourceList{
					apiv1.ResourceCPU:    resource.MustParse("50m"),
					apiv1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
		})
		// let plugin-controller know that it has started.
		// note that this hook will probably fail if the plugin container
		// runs too fast. for example, bash echo "hi" makes the plugin container
		// exits before Kubernetes attempts to inject this hook
		containers[0].Lifecycle = &apiv1.Lifecycle{
			PostStart: &apiv1.LifecycleHandler{
				Exec: &apiv1.ExecAction{
					Command: []string{"touch", "/waggle/started"},
				},
			},
		}
	}

	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: rm.labelsForPlugin(&pr.Plugin),
		},
		Spec: apiv1.PodSpec{
			ServiceAccountName: "wes-plugin-account",
			PriorityClassName:  "wes-app-priority",
			NodeSelector:       nodeSelectorForConfig(pr.Plugin.PluginSpec),
			// TODO: The priority class will be revisited when using resource metrics to schedule plugins
			// NOTE: ShareProcessNamespace allows containers in a pod to share the process namespace.
			//       containers in that pod can see other's processes
			ShareProcessNamespace: booltoPtr(true),
			InitContainers:        initContainers,
			Containers:            containers,
			Volumes:               volumes,
		},
	}, nil
}

// CreateKubernetesPod creates a Pod for the plugin
func (rm *ResourceManager) CreatePodTemplate(pr *datatype.PluginRuntime) (*apiv1.Pod, error) {
	name, err := pluginNameForSpecDeployment(&pr.Plugin)
	if err != nil {
		return nil, err
	}
	template, err := rm.createPodTemplateSpecForPlugin(pr)
	if err != nil {
		return nil, err
	}
	template.Spec.RestartPolicy = apiv1.RestartPolicyNever
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rm.Namespace,
			Labels:    template.Labels,
		},
		Spec: template.Spec,
	}, nil
}

// CreateK3SJob creates and returns a Kubernetes job object of the pllugin
func (rm *ResourceManager) CreateJob(pr *datatype.PluginRuntime) (*batchv1.Job, error) {
	name, err := pluginNameForSpecDeployment(&pr.Plugin)
	if err != nil {
		return nil, err
	}
	template, err := rm.createPodTemplateSpecForPlugin(pr)
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
func (rm *ResourceManager) CreateDeployment(pr *datatype.PluginRuntime) (*appsv1.Deployment, error) {
	name, err := pluginNameForSpecDeployment(&pr.Plugin)
	if err != nil {
		return nil, err
	}
	template, err := rm.createPodTemplateSpecForPlugin(pr)
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

func (rm *ResourceManager) CreateDaemonSet(pr *datatype.PluginRuntime) (*appsv1.DaemonSet, error) {
	name, err := pluginNameForSpecDeployment(&pr.Plugin)
	if err != nil {
		return nil, err
	}
	template, err := rm.createPodTemplateSpecForPlugin(pr)
	if err != nil {
		return nil, err
	}
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rm.Namespace,
			Labels:    template.Labels,
		},
		Spec: appsv1.DaemonSetSpec{
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

func (rm *ResourceManager) UpdatePod(pod *apiv1.Pod) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pods := rm.Clientset.CoreV1().Pods(rm.Namespace)
	if _, err := pods.Get(ctx, pod.Name, metav1.GetOptions{}); err == nil {
		_, err := pods.Update(ctx, pod, metav1.UpdateOptions{})
		return err
	} else {
		_, err := pods.Create(ctx, pod, metav1.CreateOptions{})
		return err
	}
}

func (rm *ResourceManager) CreatePod(pod *apiv1.Pod) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := rm.Clientset.CoreV1().Pods(rm.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	return err
}

func (rm *ResourceManager) UpdateDeployment(deployment *appsv1.Deployment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	deployments := rm.Clientset.AppsV1().Deployments(rm.Namespace)
	// if deployment exists, then update it, else create it
	if _, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{}); err == nil {
		_, err := deployments.Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	} else {
		_, err := deployments.Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}
}

func (rm *ResourceManager) UpdateDaemonSet(daemonSet *appsv1.DaemonSet) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	daemonSets := rm.Clientset.AppsV1().DaemonSets(rm.Namespace)
	if _, err := daemonSets.Get(ctx, daemonSet.Name, metav1.GetOptions{}); err == nil {
		_, err := daemonSets.Update(ctx, daemonSet, metav1.UpdateOptions{})
		return err
	} else {
		_, err := daemonSets.Create(ctx, daemonSet, metav1.CreateOptions{})
		return err
	}
}

func (rm *ResourceManager) RunPlugin(job *batchv1.Job) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	jobClient := rm.Clientset.BatchV1().Jobs(rm.Namespace)
	_, err := jobClient.Create(ctx, job, metav1.CreateOptions{})
	return err
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

// TerminateJob terminates the Kubernetes Job object. We set the graceperiod to 1 second to terminate
// any Job or Pod in case they do not respond to the termination signal
func (rm *ResourceManager) TerminateJob(jobName string) error {
	// This option allows us to quickly spin up the same plugin
	// The Foreground option waits until the pod deletes, which takes time
	deleteDependencies := metav1.DeletePropagationBackground
	return rm.Clientset.BatchV1().Jobs(rm.Namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(0),
		PropagationPolicy:  &deleteDependencies,
	})
}

// TerminatePod terminates the Kubernetes Pod object. We set the graceperiod to 1 second to terminate
// any Job or Pod in case they do not respond to the termination signal
func (rm *ResourceManager) TerminatePod(podName string) error {
	// This option allows us to quickly spin up the same plugin
	// The Foreground option waits until the pod deletes, which takes time
	deleteDependencies := metav1.DeletePropagationBackground
	return rm.Clientset.CoreV1().Pods(rm.Namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(0),
		PropagationPolicy:  &deleteDependencies,
	})
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

func (rm *ResourceManager) GetPodLogHandler(podName string, follow bool) (io.ReadCloser, error) {
	req := rm.Clientset.CoreV1().Pods(rm.Namespace).GetLogs(podName, &apiv1.PodLogOptions{Follow: follow})
	return req.Stream(context.TODO())
}

// GetLastPluginLog fills up given buffer with the last plugin log.
func (rm *ResourceManager) GetPodLog(podName string, lastLog []byte) (int, error) {
	logReader, err := rm.GetPodLogHandler(podName, false)
	if err != nil {
		return 0, nil
	}
	defer logReader.Close()
	totalLength := 0
	buffer := make([]byte, len(lastLog))
	for {
		n, err := logReader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				if totalLength > n {
					lastLog = append(lastLog[totalLength-n:], buffer[:n]...)
				} else {
					totalLength = n
					copy(lastLog, buffer[:n])
				}
				break
			}
		}
		copy(lastLog, buffer)
		totalLength = n
	}
	return totalLength, nil
}

func (rm *ResourceManager) GetServiceClusterIP(serviceName string, namespace string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	service, err := rm.Clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return service.Spec.ClusterIP, nil
}

// CleanUp removes all currently running jobs
func (rm *ResourceManager) ClenUp() error {
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

func (rm *ResourceManager) LaunchAndWatchPlugin(pr *datatype.PluginRuntime) {
	logger.Debug.Printf("Running plugin %q...", pr.Plugin.Name)
	pod, err := rm.CreatePodTemplate(pr)
	if err != nil {
		logger.Error.Printf("Failed to create Kubernetes Pod for %q: %q", pr.Plugin.Name, err.Error())
		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(err.Error()).AddPluginMeta(&pr.Plugin).Build())
		return
	}
	err = rm.CreatePod(pod)
	defer rm.TerminatePod(pod.Name)
	if err != nil {
		logger.Error.Printf("Failed to run %q: %q", pod.Name, err.Error())
		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason(err.Error()).AddPluginMeta(&pr.Plugin).Build())
		return
	}
	logger.Info.Printf("Plugin %q is scheduled", pod.Name)
	pr.Plugin.PluginSpec.Job = pod.Name
	// NOTE: The for loop helps to re-connect to Kubernetes watcher when the connection
	//       gets closed while the plugin is running
	for {
		watcher, err := rm.WatchPod(pod.Name, rm.Namespace, 1)
		if err != nil {
			logger.Error.Printf("Failed to watch %q. Abort the execution", pod.Name)
			rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).
				AddReason(err.Error()).
				AddPodMeta(pod).
				AddPluginMeta(&pr.Plugin).
				Build())
			return
		}
		chanEvent := watcher.ResultChan()
		defer watcher.Stop()
		for event := range chanEvent {
			logger.Debug.Printf("Plugin %s received an event %s", pod.Name, event.Type)
			_pod := event.Object.(*v1.Pod)
			switch event.Type {
			case watch.Added:
				rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusLaunched).
					AddPodMeta(_pod).
					AddPluginMeta(&pr.Plugin).
					Build())
			case watch.Modified:
				logger.Debug.Printf("%s: %s", _pod.Name, _pod.Status.Phase)
				for _, c := range _pod.Status.ContainerStatuses {
					logger.Debug.Printf("%s: (%s) %s", _pod.Name, c.Name, &c.State)
				}
				switch _pod.Status.Phase {
				case v1.PodSucceeded:
					rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusComplete).
						AddPodMeta(_pod).
						AddPluginMeta(&pr.Plugin).
						Build())
					return
				case v1.PodFailed:
					// even if the pod failed, if the plugin container succeeded
					// we consider the pod succeeded.
					pluginContainerStatus := _pod.Status.ContainerStatuses[0]
					if pluginContainerStatus.State.Terminated.ExitCode == 0 {
						// only for debugging purpose
						if pcStatus := _pod.Status.ContainerStatuses[1].State.Terminated; pcStatus != nil {
							logger.Debug.Printf("%s's plugin controller failed: %d (%s) %s", _pod.Name, pcStatus.ExitCode, pcStatus.Reason, pcStatus.Message)
						}
						rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusComplete).
							AddPodMeta(_pod).
							AddPluginMeta(&pr.Plugin).
							Build())
						return
					}
					logger.Error.Printf("Plugin %q has failed.", _pod.Name)
					eventBuilder := datatype.NewEventBuilder(datatype.EventPluginStatusFailed).
						AddPodMeta(_pod).
						AddPluginMeta(&pr.Plugin)
					if state := _pod.Status.ContainerStatuses[0].State.Terminated; state != nil {
						eventBuilder = eventBuilder.AddReason(state.Reason).
							AddEntry("return_code", fmt.Sprintf("%d", state.ExitCode))
					}
					lastLog := make([]byte, 1024)
					if lengthRead, err := rm.GetPodLog(_pod.Name, lastLog); err == nil {
						eventBuilder = eventBuilder.AddEntry("error_log", string(lastLog[:lengthRead]))
						logger.Debug.Printf("Logs of the plugin %q: %s", _pod.Name, string(lastLog[:lengthRead]))
					} else {
						eventBuilder = eventBuilder.AddEntry("error_log", err.Error())
						logger.Debug.Printf("Failed to get plugin log: %s", err.Error())
					}
					rm.Notifier.Notify(eventBuilder.Build())
					return
				}
			case watch.Deleted:
				logger.Debug.Printf("Plugin got deleted. Returning resource and notify")
				rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).
					AddReason("Plugin deleted").
					AddPodMeta(_pod).
					AddPluginMeta(&pr.Plugin).
					Build())
				return
			case watch.Error:
				logger.Debug.Printf("Error on watcher. Returning resource and notify")
				rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).
					AddReason("Error on watcher").
					AddPodMeta(_pod).
					AddPluginMeta(&pr.Plugin).
					Build())
				return
			default:
				logger.Error.Printf("Watcher of plugin %q received unknown event %q", _pod.Name, event.Type)
			}
		}
		watcher.Stop()
		logger.Error.Printf("Watcher of the plugin %s is unexpectedly closed. ", pod.Name)
		// when a pod becomes unhealthy (e.g., a host device of the pod disconnected) the watcher
		// gets closed, but the job remains valid in the cluster and never runs the pod again.
		// To get out from this loop, we check if the pod is running, if not, we should terminate the plugin
		// if pod, err := rm.GetPod(job.Name); err != nil {
		// 	logger.Error.Printf("failed to get status of pod for job %q: %s", job.Name, err.Error())
		// 	rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason("pod no longer exist").AddK3SJobMeta(job).AddPluginMeta(&pr.Plugin).Build())
		// 	return
		// } else {
		// 	if pod.Status.Phase != apiv1.PodRunning {
		// 		logger.Error.Printf("pod %q is not running for job %q. Closing plugin", pod.Name, job.Name)
		// 		rm.Notifier.Notify(datatype.NewEventBuilder(datatype.EventPluginStatusFailed).AddReason("pod no longer running").AddK3SJobMeta(job).AddPluginMeta(&pr.Plugin).AddPodMeta(pod).Build())
		// 		return
		// 	}
		// }
		logger.Info.Printf("attemping to re-connect for pod %q", pod.Name)
	}

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

func (rm *ResourceManager) RemovePlugin(p *datatype.Plugin) {
	rm.TerminateJob(p.Name)
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

func (rm *ResourceManager) Run() {
	if rm.MetricsClient == nil {
		logger.Info.Println("No metrics client is set. Metrics information cannot be obtained")
	}
	// metricsTicker := time.NewTicker(5 * time.Second)
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
			// case <-metricsTicker.C:
			// 	nodeMetrics, err := rm.MetricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
			// 	if err != nil {
			// 		logger.Error.Println("Error:", err)
			// 		return
			// 	}
			// 	for _, nodeMetric := range nodeMetrics.Items {
			// 		cpuQuantity := nodeMetric.Usage.Cpu().String()
			// 		memQuantity := nodeMetric.Usage.Memory().String()
			// 		msg := fmt.Sprintf("Node Name: %s \n CPU usage: %s \n Memory usage: %s", nodeMetric.Name, cpuQuantity, memQuantity)
			// 		logger.Debug.Println(msg)
			// 	}
			// podMetrics, err := rm.MetricsClient.MetricsV1beta1().PodMetricses(rm.Namespace).List(context.TODO(), metav1.ListOptions{})
			// 	for _, podMetric := range podMetrics.Items {
			// 		podContainers := podMetric.Containers
			// 		for _, container := range podContainers {
			// 			cpuQuantity := container.Usage.Cpu().String()
			// 			memQuantity := container.Usage.Memory().String()
			// 			msg := fmt.Sprintf("Container Name: %s \n CPU usage: %s \n Memory usage: %s", container.Name, cpuQuantity, memQuantity)
			// 			logger.Debug.Println(msg)
			// 		}
			// 	}
		}
	}
	// for {
	// nodeMetrics, err := rm.MetricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
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
	chanEvent := watcher.ResultChan()
	for {
		select {
		case e, ok := <-chanEvent:
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

func int64Ptr(i int64) *int64 { return &i }

func booltoPtr(b bool) *bool { return &b }

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
