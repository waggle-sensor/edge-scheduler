# Tutorial: specifying computing resources
This tutorial demonstrates how to request computing resource to the system. If not specified, the Waggle node allocates resources to plugins using the default as follows (See [wes-default-limits](https://github.com/waggle-sensor/waggle-edge-stack/blob/main/kubernetes/wes-default-limits.yaml)),

```yaml
...
  limits:
    - default:
        memory: 1Gi
      defaultRequest:
        cpu: 500m
        memory: 300Mi
...
```

## Available Resource Types
- request.cpu: The amount of CPU resource to request. The plugin will be guaranteed to get the amount when successfully scheduled and run.
- request.memory: The amount of Memory resource to request. The plugin will be guaranteed to get the amount when successfully scheduled and run.
- limit.cpu: The amount of CPU that cannot be exceeded by the plugin. When exceeds, the plugin will start thottling.
- limit.memory: The amount of Memory that cannot be exceeded by the plugin. Note that this has to be greater than the plugins' workingset memory amount, otherwise the plugin will be out-of-memory killed.

Note that if the request amount is larger than the system can provide, the plugin will be in the pending state until the requested amount is available. In general, do not request more than the plugins would use (i.e., actual consumption +- 10%).

The rules are,
- The CPU value should be either an integer or a value in millicore. Examples include 1 (1 logical CPU core), 500m (half of one logical CPU core). 1000m equals to 1.
- The memory value should be a value with units such as Mi, Gi, and so on. Examples are 100Mi, 2Gi.
- limit.# types must have a matching request.#. For example, request.cpu=1,limit.cpu=2. Otherwise the system will throw an error
- The amount of limit must be greater than the amount of request. For example, request.cpu=1,limit.cpu=500m will not be accepted
- List the resource types in a comma-separated string (see the example below)

```bash
pluginctl deploy --name plugin-objectcounter --resource request.cpu=1,limit.cpu=2,request.memory=2Gi,limit.memory=3Gi --selector resource.gpu=true sagecontinuum.org/yonghokim/object-counter:0.5.1 -- -stream bottom_camera -all-objects -continuous
```