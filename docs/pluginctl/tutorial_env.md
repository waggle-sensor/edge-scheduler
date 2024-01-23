# Tutorial: setting environmental variables to plugin
This tutorial demonstrates the ways to set environmental variables to plugins.

An environmental variable can be set,

```bash
pluginctl run \
  --name test \
  --entrypoint bash \
  --env FOO=BAR \
  waggle/plugin-base:1.1.1-base \
  -- \
  -c 'printenv | grep FOO'
```

The output would look like,
```bash
Scheduled the plugin test successfully 
INFO: 2024/01/23 16:47:22 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 16:47:24 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 16:47:26 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 16:47:28 run.go:78: Plugin is in "Pending" state. Waiting...
FOO=BAR
```

__NOTE: You may need to kill the plugin by pressing Ctrl+C if it does not exit__

One advanced use case is to set the variable value from an external source. We support importing Kubernetes Secret literal as a value to the environmental variable. This approach may require you to ssh into the node and have the `kubectl` command executable from you.

First, let's create a Kubernetes Secret with a literal,

```bash
sudo kubectl create secret generic mysecret \
  --from-literal=MYFOO=MYBAR
```

The output is,

```bash
secret/mysecret created
```

Then, run the plugin again by follows,

```bash
pluginctl run \
  --name test \
  --entrypoint bash \
  --env FOO={secret.mysecret.MYFOO} \
  waggle/plugin-base:1.1.1-base \
  -- \
  -c 'printenv | grep FOO'
```

Output would look like,

```bash
Scheduled the plugin test successfully 
INFO: 2024/01/23 17:02:23 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 17:02:25 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 17:02:27 run.go:78: Plugin is in "Pending" state. Waiting...
INFO: 2024/01/23 17:02:29 run.go:78: Plugin is in "Pending" state. Waiting...
FOO=MYBAR
```

__NOTE: You may need to kill the plugin by pressing Ctrl+C if it does not exit__

The `{secret.mysecret.MYFOO}` tells pluginctl to get the value `MYBAR` from the literal `MYFOO` in the Secret `mysecret`.

You can edit the Secret to change the value by,

```bash
sudo kubectl edit mysecret
```

But, the change does not get updated on plugins that are already running. In order to reflect the change in the plugin, you have to restart the plugin.