# States of a Plugin over its lifecycle

<!-- for the overall doc -->
An Edge aplication is selected by a job when submitted to the system. The application is then implemented with a wrapper called Plugin and runs on Waggle nodes. The Waggle scheduler records states of a Plugin over its lifecycle to inform users how the application runs.

Plugin state changes based on events received from the Waggle edge scheduler and the backend Kubernetes cluster in Waggle node. The following table lists the states along with description for each state.

| State    | Description |
| :--------: | :------- |
| queued | The Plugin has been triggered by its science Rules and placed in the queue for execution. It will be dequeued by the scheduler as the required resource can be allocated by the system. |
| selected | The Plugin is selected by scheduling policy and dequeued. Scheduler-estimated resource availability shows that the Plugin can be scheduled. |
| scheduled | The resource is allocated to the Plugin and associated container (i.e., a Kubernetes Pod) is created. This does not mean the Plugin actually runs its program, but indicates the creation of the program container. |
| initializing | The Plugin is being initialized. This initialzation includes container image pulling if not exist on the node, connecting the Plugin to the Waggle data pipeline, etc. |
| running | The Plugin program starts to run on designiated device. |
| completed | The Plugin program is terminated successfully, i.e. receiving return code 0 from the program container. |
| failed | The Plugin failed to reach to "completed" state. There are various reasons that a Plugin would end up with this state. For example, Plugin may fail to initialize and it will transition to this state with an error of the initialization. Or, Plugin code exited with non-zero return code.

# Other useful events
In addition to the states reported by the Waggle edge scheduler, it reports other events to aid users in understanding the Plugin execution deeper. The common events include creation of containers, pulling containers from remote/local registries, etc. In combination with the Plugin states, this can provide in-depth information of how Plugins run.

# Event change logs
This section is to keep the changes in history such that anyone parsing the events can correctly interpret them. By tracking [the version change](https://github.com/waggle-sensor/waggle-edge-stack/blob/main/kubernetes/wes-plugin-scheduler.yaml#L33) in the scheduler, one should be able to correlate scheduling events with this document.

## Version 0.26.0 and prior

| State    | Description |
| :--------: | :------- |
| promoted | The Plugin has been triggered by its science Rules and placed in the queue for execution. It will be dequeued by the scheduler as the required resource can be allocated by the system. |
| launched | The resource is allocated to the Plugin and associated container (i.e., a Kubernetes Pod) is created. This does not mean the Plugin actually runs its program, but indicates the creation of the program container. |
| complete | The Plugin program is terminated successfully, i.e. receiving return code 0 from the program container. |
| failed | The Plugin failed to reach to "completed" state. There are various reasons that a Plugin would end up with this state. For example, Plugin may fail to initialize and it will transition to this state with an error of the initialization. Or, Plugin code exited with non-zero return code.

# Developer notes

## Inactive state of plugin
The following state may need to be considered in the future if we need to capture this state.

| State    | Description |
| :--------: | :------- |
| inactive  | The Plugin is recognized by the Waggle edge scheduler on a Waggle node. In this state, the scheduler constantly evaluates any associated science rules to check if the Plugin needs to run to capture any event of interest. |