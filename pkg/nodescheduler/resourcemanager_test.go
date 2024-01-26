package nodescheduler

import (
	"context"
	"testing"
	"time"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestEnvFromSecret(t *testing.T) {
	mySecretWord := "helloworld"
	// A secret which we will retrieve the data from.
	objects := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysecret",
				Namespace: "ses", // ResourceManager uses ses namespace by default
			},
			Data: map[string][]byte{
				"greeting": []byte(mySecretWord),
			},
		},
	}
	fake_rm := NewFakeK3SResourceManager(objects)

	pr := datatype.NewPluginRuntimeWithScienceRule(
		datatype.Plugin{
			Name: "test-plguin",
			PluginSpec: &datatype.PluginSpec{
				Image: "myimage:latest",
				Env: map[string]string{
					"ENV1":  "123",
					"MYENV": "{secret.mysecret.greeting}",
				},
			},
		}, datatype.ScienceRule{},
	)
	pluginSpec, err := fake_rm.createPodTemplateSpecForPlugin(pr)
	if err != nil {
		t.Error(err)
	}
	pluginContainer := pluginSpec.Spec.Containers[0]
	found := false
	for _, e := range pluginContainer.Env {
		if e.Name == "MYENV" && e.Value == mySecretWord {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("%s was expected in %v, but not found", mySecretWord, pluginContainer.Env)
	}
	out, _ := yaml.Marshal(pluginSpec)
	t.Logf("%s", out)
}

// TestFakeClient demonstrates how to use a fake client with SharedInformerFactory in tests.
func TestFakeClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherStarted := make(chan struct{})
	// Create the fake client.
	client := fake.NewSimpleClientset()
	// A catch-all watch reactor that allows us to inject the watcherStarted channel.
	client.PrependWatchReactor("*", func(action clienttesting.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := client.Tracker().Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		close(watcherStarted)
		return true, watch, nil
	})

	// We will create an informer that writes added pods to a channel.
	pods := make(chan *v1.Pod, 1)
	informers := informers.NewSharedInformerFactory(client, 0)
	podInformer := informers.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			t.Logf("pod added: %s/%s", pod.Namespace, pod.Name)
			pods <- pod
		},
	})

	// Make sure informers are running.
	informers.Start(ctx.Done())

	// This is not required in tests, but it serves as a proof-of-concept by
	// ensuring that the informer goroutine have warmed up and called List before
	// we send any events to it.
	cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced)

	// The fake client doesn't support resource version. Any writes to the client
	// after the informer's initial LIST and before the informer establishing the
	// watcher will be missed by the informer. Therefore we wait until the watcher
	// starts.
	// Note that the fake client isn't designed to work with informer. It
	// doesn't support resource version. It's encouraged to use a real client
	// in an integration/E2E test if you need to test complex behavior with
	// informer/controllers.
	<-watcherStarted
	// Inject an event into the fake client.
	p := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}}
	_, err := client.CoreV1().Pods("test-ns").Create(context.TODO(), p, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error injecting pod add: %v", err)
	}

	select {
	case pod := <-pods:
		t.Logf("Got pod from channel: %s/%s", pod.Namespace, pod.Name)
	case <-time.After(wait.ForeverTestTimeout):
		t.Error("Informer did not get the added pod")
	}
}
