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
				Name:       "testing",
				Args:       []string{"1", "2", "3"},
				Privileged: true,
				Image:      "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-b6b14d4c",
		},
		{
			spec: &Spec{
				Name:       "testing",
				Args:       []string{"1", "2", "3"},
				Privileged: false,
				Image:      "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-e11b3468",
		},
		{
			spec: &Spec{
				Name:       "testing",
				Args:       []string{"1", "2", "3"},
				Privileged: false,
				Image:      "waggle/cool-plugin:1.2.3",
				Node:       "rpi-1",
			},
			want: "cool-plugin-1-2-3-f67182a5",
		},
		{
			spec: &Spec{
				Name:       "testing",
				Args:       []string{"1", "2", "3"},
				Privileged: false,
				Image:      "waggle/cool-plugin:1.2.3",
				Node:       "nx-1",
			},
			want: "cool-plugin-1-2-3-2369ef48",
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s, err := generatePluginNameForSpec(tc.spec)
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
