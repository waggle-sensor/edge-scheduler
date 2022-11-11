# Scheduling Policy

The scheduler will need to adapt different scheduling policies in order to accommodate different scheduling requirements. An abstract template is given for scheduling policy creators to follow such that they can be natively implemented in the scheduler.

# Policy Template

Scheduling policy will need to implment the following method(s) in its class.

```go
// https://github.com/waggle-sensor/edge-scheduler/blob/main/pkg/nodescheduler/policy/default.go#L8
SelectBestPlugins(*datatype.Queue, *datatype.Queue, datatype.Resource) ([]*datatype.Plugin, error)
```
The function should return a list of plugins that the policy selects as the best plugins to run at any given time. The scheduler calls this function whenever resource is available. The list is ordered such that plugins in the earier index in the list means higher priority over the plugins in the later index.

# Add a scheduling policy

Once