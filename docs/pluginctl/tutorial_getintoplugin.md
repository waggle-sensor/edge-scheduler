# Tutorial: getting into a plugin
This tutorial creates and runs a plugin. Then, it allows you to get into the plugin container to execute commands interactively.

We first use `deploy` to run a plugin named "getintoplugin" with the arguments that has an infinite loop to sleep. This sleep is needed for us to buy time while we are executing commands inside the plugin. Without this sleep mechanism, the plugin would execute default command which does nothing and terminate almost immediately.
```bash
pluginctl deploy --name getintoplugin waggle/plugin-base:1.1.1-base -- bash -c 'while true; do sleep 1; done'
```

The output would be,
```bash
Launched the plugin getintoplugin successfully 
You may check the log: pluginctl logs getintoplugin
To terminate the job: pluginctl rm getintoplugin
```

We then run `ps` subcommand to check if the plugin is running. If it is not in running state, the plugin image might be being downloaded from Internet, if the system does not have the image cached.
```bash
pluginctl ps
```

The output should look like,
```bash
NAME               STATUS      START_TIME                RUNNING_TIME
getintoplugin   Running     2022/12/02 22:03:50 UTC   6.906323958s
```

Now we run `exec` subcommand with `-ti` options indicating that we attach your terminal to the shell inside the plugin container. This allows you to use your terminal to execute commands inside the container. We then execute date and echo commands there, see the result, and exit. This exit exits the shell and detaches the terminal from the shell.
```bash
pluginctl exec -ti getintoplugin -- /bin/bash
root@getintoplugin-nfrcj:/# date
Wed Jan 19 19:51:05 UTC 2022
root@getintoplugin-nfrcj:/# echo "I am inside container"
I am inside container
root@getintoplugin-nfrcj:/# exit
exit
```

We have one last thing before closing this tutorial, clean up. Because the container would run forever because of the infinite loop and sleep, we need to terminate the plugin. The `rm` subcommand deletes the container from the system.
```bash
pluginctl rm getintoplugin
```
The output should be,
```bash
Terminated the plugin getintoplugin successfully
```
