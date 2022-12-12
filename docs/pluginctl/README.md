# Command-line tool for running plugin
Pluginctl is a command-line tool to run edge applications (i.e., plugins) for development and testing. The tool requires a kubeconfig for cluster access (by default use `${USER}/.kube/config`) inside a Waggle node.

```bash
pluginctl --kubeconfig /PATH/TO/KUBECONFIG
```

The output would be,
```bash
SAGE plugin control for running plugins: 0.18.0
pluginctl --help for more information
```

# Installation and set up
Pluginctl supports for amd64 and arm64 architecture. Please check [Releases](https://github.com/waggle-sensor/edge-scheduler/releases) to download the binary for Waggle/Sage node or talk to the system manager to get the tool installed. The tool may not work properly if Kubernetes does not have [Waggle edge stack](https://github.com/waggle-sensor/waggle-edge-stack) (WES) installed and running. Sage/Waggle nodes already have WES running at the system level.

```bash
# For arm64 architecture
wget -O /usr/bin/pluginctl https://github.com/waggle-sensor/edge-scheduler/releases/download/0.18.2/pluginctl-linux-arm64
chmod +x /usr/bin/pluginctl
```

# Tutorials
The tutorials demonstrate how to use `pluginctl` on Waggle/Sage nodes to test, debug, and finalize plugin development.

1. [Hello World](tutorial_helloworld.md) shows how to create and run a plugin container

2. [Getting into plugin](tutorial_getintoplugin.md) lets you get into a plugin container for development and testing

3. [Printing logs](tutorial_printlog.md) prints logs of a plugin

4. [Build plugin](tutorial_build.md) builds a plugin from AI/ML code hosted on a public code repository

5. [Profiling Edge Applications](tutorial_profiling.md) runs an AI plugin and saves plugin performance data into a file after the run