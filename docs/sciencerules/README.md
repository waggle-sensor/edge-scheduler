# Driving node-level behaviors using science rule
The node scheduler running on Waggle nodes takes user jobs to schedule edge applications. To best serve user intention on scheduling the applications, it is important to take user requirement written in a form that the scheduler can interpret and use.

A science rule consists of 2 parts: condition and action. It simply means "when the condition is valid, perform the aciton". The syntax of a science rule is,
```
<<action>> : <<condition>>
```

We envision that by using science rules end users should be able to manipulate node-level behaviors that can possibly trigger cloud and/or intra-node level behaviors. For example, one Waggle node scheduling a plugin and reporting an important data back to the cloud can trigger the cloud to run a corresponding simulation in high performance computing, and that will feed back to the node with a new behavior driving the node to observe the environment differently.

# Actions in science rule
Science rule supports 3 different actions to perform: `schedule`, `publish`, and `set`.

1. `schedule` simply tell the node scheduler to schedule plugin. The specified name of the plugin must match with the plugin name of `plugins` specified in a job description.
```bash
# schedule myplugin whenever the node can
schedule(myplugin): True
```

2. `publish` publishes a message to the cloud. This is useful when we need a node-to-cloud trigger from locally measured data by plugins,
```bash
# we assume myplugin publishes env.car.crashed whenever it detects a car accident from node's camera
schedule(myplugin): True
publish(env.event.car_accident): any(v('env.car.crashed', since='-1m'))
```
`publish` also publishes the message to local node for other plugins to receive in case the plugins need to react on certain events.

3. `set` sets a state to node's local storage. This is useful when users want the plugin to behave differently, without changing plugin logic too much.
```bash
# schedul myplugin every 5 minutes
# we set the variable rainy to 1 if it has rained more than 3 mm / hour, otherwise we set it to 0
# myplugin checks the local state "rainy" and changes
# its behavior for better data measurement
schedule(myplugin): cronjob("myplugin", "*/5 * * * *")
set(rainy, value=1): sum(rate('env.raingauge.total_acc', since="-1h") > 3.
set(rainy, value=0): sum(rate('env.raingauge.total_acc', since="-1h") <= 3.
```

# Conditions in science rule
The condition is evaluated by the Python3 engine. Therefore, any Python3-formatted condition can be properly evaluated.

Below rule is always valid as the condition is always evaluated as True by Python3,
```python
# if 1 + 2 == 3, then schedule myplugin
schedule(myplugin): 1 + 2 == 3
```

However, having one line Python script is not flexible enough for users to describe complex conditions that best capture their requirement. For example, the following user requirement needs the `v` function to get locally perceived temperature values,
```python
# if the averaged temperature in the last minute is greater than 30, then schedule myplugin
# we describe the rule using the v function we developed
schedule(myplugin): avg(v('env.temperature')) > 30.0
```

To support such detailed science rules, we have created [supported functions](https://github.com/waggle-sensor/sciencerule-checker/blob/master/docs/supported_functions.md) for users to use.