package cloudscheduler

import (
	"io/ioutil"
	"testing"
)

func TestWhiteList(t *testing.T) {
	v := NewJobValidator(&CloudSchedulerConfig{DataDir: "/tmp"})
	l := `^waggle/(.*)
^registry.github.com/waggle/yourimage:1.(.*)
^registry.gitlab.com/waggle/(.*)
^registry.gitlab.com/collab/myimage:1.2.[1-9+]
`
	ioutil.WriteFile("/tmp/plugins.whitelist", []byte(l), 0644)
	v.LoadPluginWhitelist()
	tests := map[string]struct {
		Input string
		Wants bool
	}{
		"test 1": {
			Input: "waggle/myimage:1.2.3",
			Wants: true,
		},
		"test 2": {
			Input: "wagglea/myimage:1.2.3",
			Wants: false,
		},
		"test 3": {
			Input: "docker.io/waggle/myimage:1.2.3",
			Wants: false,
		},
		"test 4": {
			Input: "registry.gitlab.com/waggle/myimage:1.2.3",
			Wants: true,
		},
		"test 5": {
			Input: "registry.github.com/waggle/myimage:1.2.3",
			Wants: false,
		},
		"test 6": {
			Input: "registry.github.com/waggle/yourimage:2.3.4",
			Wants: false,
		},
		"test 7": {
			Input: "registry.github.com/waggle/yourimage:1.12.13",
			Wants: true,
		},
		"test 8": {
			Input: "registry.gitlab.com/collab/myimage:1.3.3",
			Wants: false,
		},
		"test 9": {
			Input: "registry.github.com/waggle/yourimage:1.2.3",
			Wants: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if r := v.IsPluginWhitelisted(tc.Input); r != tc.Wants {
				t.Fatalf("expected %t, but received %t for plugin %q", tc.Wants, r, tc.Input)
			}
		})
	}
}

func TestPullPluginManigest(t *testing.T) {
	v := NewJobValidator(&CloudSchedulerConfig{
		DataDir: "/tmp",
		ECRURL:  "https://ecr.sagecontinuum.org",
	})
	_, err := v.AttemptToFindPluginManifest("myregistry.mycompany.com/my/plugin:0.1.2")
	if err == nil {
		t.Fatalf("Test did not fail which should")
	}
	_, err = v.AttemptToFindPluginManifest("yonghokim/object-counter:0.5.1")
	if err != nil {
		t.Fatalf("%s", err.Error())
	}
}
