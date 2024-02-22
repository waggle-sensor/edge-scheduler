# States of a Plugin over its lifecycle

<!-- for the overall doc -->
An Edge aplication is selected by a job when submitted to the system. The application is then implemented with a wrapper called Plugin and runs on Waggle nodes. The Waggle scheduler records states of a Plugin over its lifecycle to inform users how the application runs.

Plugin state changes based on events received from the Waggle edge scheduler and the backend Kubernetes cluster in Waggle node. The following table lists states of a Plugin along with description for each state.

| State    | Description |
| :--------: | :------- |
| inactive  | The Plugin is recognized by the Waggle edge scheduler on a Waggle node. In this state, the scheduler constantly evaluates any associated Science rules to check if the Plugin needs to run to capture any event of interest. |
| queued | The Plugin has been triggered by its Science Rules and placed in the queue for execution. It will be dequeued by the scheduler as the required resource can be allocated by the system. |
| scheduled | The resource is allocated to the Plugin and associated container (i.e., a Kubernetes Pod) is created. This does not mean the Plugin actually runs its program, but indicates the creation of the program container. |
| initializing | The Plugin is being initialized. This initialzation includes container image pulling if not exist on the node, connecting the Plugin to the Waggle data pipeline, etc. |
| running | The Plugin program starts to run on designiated device. |
| completed | The Plugin program is terminated successfully, i.e. receiving return code 0 from the program container. |
| failed | The Plugin failed to reach to "completed" state. There are various reasons that a Plugin would end up with this state. For example, Plugin may fail to initialize and it will transition to this state with an error of the initialization. Or, Plugin code exited with non-zero return code.