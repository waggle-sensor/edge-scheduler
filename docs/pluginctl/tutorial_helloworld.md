# Tutorial: Hello World 
In this tutorial we use pluginctl to run a container and print "hello world" from inside of the container.

```bash
$ pluginctl run --name helloworld waggle/plugin-base:1.1.1-base -- bash -c 'echo "hello world"'
Launched the plugin helloworld-1642620725-1642620725 successfully 
INFO: 2022/01/19 19:32:05 run.go:57: Plugin is in "Pending" state. Waiting...
INFO: 2022/01/19 19:32:07 run.go:57: Plugin is in "Pending" state. Waiting...
hello world
```

The `run` subcommand runs the Docker image (waggle/plugin-base:1.1.1-base) with the arguments specified after `--`. The arguments were executed inside container and resulted in echoing "hello world". The Docker image must be available to the node. The image used in this tutorial is available [here](https://hub.docker.com/r/waggle/plugin-base). 