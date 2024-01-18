package nodescheduler

import (
	"reflect"
	"regexp"
	"testing"
)

func TestEnvFromSecret(t *testing.T) {
	pluginEnvFromSecretRegexFormat = regexp.MustCompile(`^{secret\.([a-z0-9-]+).([a-zA-Z0-9]+)}$`)

	kv := map[string]string{
		"ENV1":                    "123",
		"{secret.mysecret.myenv}": "helloworld",
	}
	expected := map[string]string{
		"ENV1":  "123",
		"myenv": "helloworld",
	}
	parsed := make(map[string]string)
	for k, v := range kv {
		// if the environment variable name refers to a Kubernetes Secret,
		// then we check and load the Secret from Kubernetes.
		// Else, the it may be just a variable name
		// FYI, len(sp) must be 3; [{secret.mysecret.myenv} mysecret myenv]
		if sp := pluginEnvFromSecretRegexFormat.FindStringSubmatch(k); len(sp) == 3 {
			// secretName := sp[1]
			keyName := sp[2]
			// fmt.Printf("%v and name is %q, env name is %q", sp, secretName, keyName)
			parsed[keyName] = v
		} else {
			parsed[k] = v
		}
	}
	if !reflect.DeepEqual(parsed, expected) {
		t.Errorf("%+v but expected %+v", parsed, expected)
	}
}
