# Tutorial: hello world 
In this tutorial we use pluginctl to run a plugin container and print "hello world" from inside of the container.


The `run` subcommand runs the plugin container image (waggle/plugin-base:1.1.1-base) with the arguments specified after `--`.
```bash
pluginctl run --name helloworld waggle/plugin-base:1.1.1-base -- bash -c 'echo "hello world"'
```

The arguments were executed inside container and resulted in echoing "hello world",
```bash
Scheduled the plugin helloworld successfully 
INFO: 2022/12/02 21:59:56 run.go:75: Plugin is in "Pending" state. Waiting...
INFO: 2022/12/02 21:59:58 run.go:75: Plugin is in "Pending" state. Waiting...
hello world
```

The container image must be available to the node. The image used in this tutorial is available in [Docker hub](https://hub.docker.com/r/waggle/plugin-base).
