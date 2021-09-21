package runplugin

import (
	"fmt"
	"regexp"
	"testing"
)

func TestGeneratedName(t *testing.T) {
	tests := []struct {
		spec *Spec
		want string
	}{
		{
			spec: &Spec{
				Args:  []string{"1", "2", "3"},
				Image: "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-1e463618",
		},
		{
			spec: &Spec{
				Args:       []string{"1", "2", "3"},
				Privileged: true,
				Image:      "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-77274c42",
		},
		{
			spec: &Spec{
				Privileged: true,
				Image:      "waggle/sensor-plugin:0.4.1",
				Node:       "rpi-1",
				Job:        "weather",
			},
			want: "sensor-plugin-0-4-1-cb0c0269",
		},
		{
			spec: &Spec{
				Args:       []string{"--debug"},
				Privileged: true,
				Image:      "waggle/sensor-plugin:0.4.1",
				Node:       "nx-1",
				Job:        "weather",
			},
			want: "sensor-plugin-0-4-1-05d08623",
		},
		{
			spec: &Spec{
				Name:       "custom-plugin-name",
				Args:       []string{"1", "2", "3"},
				Privileged: false,
				Image:      "waggle/cool-plugin:1.2.3",
				Node:       "nx-1",
			},
			want: "custom-plugin-name",
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s, err := pluginNameForSpec(tc.spec)
			if err != nil {
				t.Fatalf("error: %s", err.Error())
			}
			if s != tc.want {
				t.Fatalf("expected: %v, got: %v", tc.want, s)
			}
		})
	}
}

func TestValidPluginNames(t *testing.T) {
	tests := map[string]struct {
		spec *Spec
		want *regexp.Regexp
	}{
		"simple": {
			spec: &Spec{
				Image: "plugin-iio:0.2.0",
			},
			want: regexp.MustCompile("^plugin-iio-0-2-0-[0-9a-f]{8}$"),
		},
		"version": {
			spec: &Spec{
				Image: "plugin-metsense:1.2.3",
			},
			want: regexp.MustCompile("^plugin-metsense-1-2-3-[0-9a-f]{8}$"),
		},
		"named": {
			spec: &Spec{
				Name:  "named-hello-world",
				Image: "plugin-metsense:1.2.3",
			},
			want: regexp.MustCompile("^named-hello-world$"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pluginName, err := pluginNameForSpec(tc.spec)
			if err != nil {
				t.Fatalf("error: %s", err.Error())
			}
			if !tc.want.MatchString(pluginName) {
				t.Fatalf("expected: %v, got: %v", tc.want, pluginName)
			}
		})
	}
}

func TestInvalidPluginNames(t *testing.T) {
	tests := map[string]struct {
		spec *Spec
	}{
		"uppercase": {
			spec: &Spec{
				Name:  "plugin-named-hello-World",
				Image: "plugin-metsense:1.2.3",
			},
		},
		"symbol": {
			spec: &Spec{
				Name:  "plugin-named-hello-@world",
				Image: "plugin-metsense:1.2.3",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := pluginNameForSpec(tc.spec)
			if err == nil {
				t.Fatalf("expected error for test %v spec: %v", name, tc.spec)
			}
		})
	}
}
