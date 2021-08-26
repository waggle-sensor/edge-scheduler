package runplugin

import (
	"regexp"
	"testing"
)

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
