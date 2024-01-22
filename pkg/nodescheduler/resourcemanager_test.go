package nodescheduler

import (
	"testing"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
