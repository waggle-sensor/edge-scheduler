# Tutorial: performance profiling of a plugin
This tutorial runs an AI plugin and aggregates performance data of the plugin after the run. This will not only show resource footprint of the plugin, but also help plugin developers understand the resource requirement of the plugin on a Waggle node.

To profile a plugin,

__NOTE: We need to use --selector to specify that the plugin requires GPU access to run the machine learning model__
```bash
pluginctl profile run --name profilingplugin --selector resource.gpu=true registry.sagecontinuum.org/yonghokim/object-counter:0.5.1 -- -stream bottom
```

The plugin will be scheduled and soon launched. It will output the result and be terminated. Then, the subcommand will attempt to get performance data of the plugin from the local database,
```bash
Launched the plugin profilingplugin successfully 
...
INFO: 2022/12/06 17:06:43 profile.go:133: Plugin took 5m39.565337679s to finish
INFO: 2022/12/06 17:06:43 pluginctl.go:393: Start gathering performance data...
```

The performance data are saved in a file named `profilingplugin.csv`. The file has multiple CSV tables containing performance metrics produced by [cadvisor](https://github.com/google/cadvisor). The metrics include CPU, memory, file read/write, and network activities. [GPU metrics](https://github.com/waggle-sensor/jetson-exporter/blob/main/README.md#metrics) are presented as a CSV table in the same file.

You can find how to interpret the metrics and plot them in the [IPython notebook](../../scripts/analysis/Analyze_plugin_performance.ipynb).