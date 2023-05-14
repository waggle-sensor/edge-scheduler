package policy

import (
	"testing"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

func TestGPUAwarePolicy(t *testing.T) {
	var (
		readyQueue       datatype.Queue
		scheduledPlugins datatype.Queue
	)
	readyQueue.Push(&datatype.PluginRuntime{
		Plugin: datatype.Plugin{
			Name: "nongpu-plugin-a",
			PluginSpec: &datatype.PluginSpec{
				Image: "plugin-a:latest",
			},
		},
	})
	readyQueue.Push(&datatype.PluginRuntime{
		Plugin: datatype.Plugin{
			Name: "gpu-plugin-b",
			PluginSpec: &datatype.PluginSpec{
				Image: "plugin-b:latest",
				Selector: map[string]string{
					"resource.gpu": "true",
				},
			},
		},
	})
	readyQueue.Push(&datatype.PluginRuntime{
		Plugin: datatype.Plugin{
			Name: "nongpu-plugin-c",
			PluginSpec: &datatype.PluginSpec{
				Image: "plugin-c:latest",
			},
		},
	})
	readyQueue.Push(&datatype.PluginRuntime{
		Plugin: datatype.Plugin{
			Name: "gpu-plugin-d",
			PluginSpec: &datatype.PluginSpec{
				Image: "plugin-d:latest",
				Selector: map[string]string{
					"resource.gpu": "true",
				},
			},
		},
	})
	schedulingPolicy := GetSchedulingPolicyByName("gpuaware")
	pluginsToSchedule, err := schedulingPolicy.SelectBestPlugins(
		&readyQueue,
		&scheduledPlugins,
		datatype.Resource{
			CPU:       "999000m",
			Memory:    "999999Gi",
			GPUMemory: "999999Gi",
		})
	if err != nil {
		t.Error(err)
	}
	if len(pluginsToSchedule) != 3 {
		t.Errorf("all 3 plugins are expected to be scheduled, but %d plugins were scheduled", len(pluginsToSchedule))
		for _, pr := range pluginsToSchedule {
			t.Log(pr.Plugin.Name)
		}
	}
}
