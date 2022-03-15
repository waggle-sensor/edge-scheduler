package datatype

import "testing"

func TestResourceConversion(t *testing.T) {
	tests := map[string]struct {
		input *Resource
		want  struct {
			CPU       int
			Memory    int
			GpuMemory int
		}
	}{
		"cpuConversion1": {
			input: &Resource{
				CPU:       "1000m",
				Memory:    "8Gi",
				GPUMemory: "8000Mi",
			},
			want: struct {
				CPU       int
				Memory    int
				GpuMemory int
			}{
				CPU:       1000,
				Memory:    8 * 1024,
				GpuMemory: 8000,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.input.convert()
			if tc.input.cpuInMilli != tc.want.CPU {
				t.Fatalf("Failed to convert CPU: expected %d, but %d", tc.want.CPU, tc.input.cpuInMilli)
			}
			if tc.input.memInMega != tc.want.Memory {
				t.Fatalf("Failed to convert Memory: expected %d, but %d", tc.want.Memory, tc.input.memInMega)
			}
			if tc.input.gpuMemInMega != tc.want.GpuMemory {
				t.Fatalf("Failed to convert Memory: expected %d, but %d", tc.want.GpuMemory, tc.input.gpuMemInMega)
			}

		})
	}
}
