# Command-line Tool For Running Plugin
Pluginctl is a command-line tool to run edge applications (i.e., plugins) for development and testing. The tool requires a kubeconfig for cluster access (by default use `${USER}/.kube/config`).

```bash
$ pluginctl --kubeconfig /PATH/TO/KUBECONFIG
SAGE edge scheduler client version: 0.8.3
pluginctl --help for more information
```

# Installation and Set up
Pluginctl supports for amd64 and arm64 architecture. Please check [Releases](https://github.com/waggle-sensor/edge-scheduler/releases) to download the binary for Waggle/Sage node or talk to the system manager to get the tool installed. The tool may not work properly if Kubernetes does not have [Waggle edge stack](https://github.com/waggle-sensor/waggle-edge-stack) (WES) installed and running. Sage/Waggle nodes already have WES running at the system level.

```bash
# For arm64 architecture
wget -O /usr/bin/pluginctl https://github.com/waggle-sensor/edge-scheduler/releases/download/0.8.3/pluginctl-arm64
chmod +x /usr/bin/pluginctl
```

# Tutorials
The tutorials show how to use `pluginctl` on Waggle/Sage nodes to test, debug, and finalize plugin development.

1. [Hello World](tutorial_helloworld.md) shows how to create and run a Docker container

2. [Getting into Container](tutorial_getintocontainer.md) lets you get into a container for development and testing

3. [Printing Logs](tutorial_printlog.md) prints logs of a container
