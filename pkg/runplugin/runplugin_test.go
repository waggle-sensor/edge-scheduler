package runplugin

import (
	"fmt"
	"testing"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

func TestGeneratedName(t *testing.T) {
	tests := []struct {
		spec *datatype.PluginSpec
		want string
	}{
		{
			spec: &datatype.PluginSpec{
				Args:  []string{"1", "2", "3"},
				Image: "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-37403588",
		},
		{
			spec: &datatype.PluginSpec{
				Args:       []string{"1", "2", "3"},
				Privileged: true,
				Image:      "waggle/cool-plugin:1.2.3",
			},
			want: "cool-plugin-1-2-3-2c591e14",
		},
		{
			spec: &datatype.PluginSpec{
				Privileged: true,
				Image:      "waggle/sensor-plugin:0.4.1",
				Node:       "rpi-1",
				Job:        "weather",
			},
			want: "sensor-plugin-0-4-1-61e6962d",
		},
		{
			spec: &datatype.PluginSpec{
				Args:       []string{"--debug"},
				Privileged: true,
				Image:      "waggle/sensor-plugin:0.4.1",
				Node:       "nx-1",
				Job:        "weather",
			},
			want: "sensor-plugin-0-4-1-c02bb84a",
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
