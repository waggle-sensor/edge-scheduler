# pluginctl - A CLI tool for running edge plugins

pluginctl is a CLI tool for running edge plugins for development and testing.

To support the most common use case of doing remote plugin development on a Sage node, pluginctl _comes preinstalled on Sage nodes. In this case, please continue to the [usage](#Usage) section._

## Installation (Advanced)

pluginctl uses a Kubernetes cluster to run jobs. Before proceeding, please ensure you have `kubectl` access to a cluster.

Visit the [latest release page](https://github.com/sagecontinuum/ses/releases/latest), download the `pluginctl` executable appropriate for your architecture as `/usr/bin/pluginctl` and mark this executable.

For example, if the latest is 0.9.3 and you are on arm64, you would run:

```bash
wget https://github.com/sagecontinuum/ses/releases/download/0.9.3/pluginctl-arm64 -O /usr/bin/pluginctl
chmod +x /usr/bin/pluginctl
```

## Usage

The following usage tutorials show how to use `pluginctl` on Waggle / Sage nodes to test, debug, and finalize plugin development.

1. [Hello World](tutorial_helloworld.md) shows how to create and run a Docker container

2. [Getting into Container](tutorial_getintocontainer.md) lets you get into a container for development and testing

3. [Printing Logs](tutorial_printlog.md) prints logs of a container
