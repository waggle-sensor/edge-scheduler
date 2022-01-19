# Tutorial: Getting into Container
This tutorial creates and runs a container and allow you to get into the container to execute commands interactively.

```bash
$ pluginctl deploy --name getintocontainer waggle/plugin-base:1.1.1-base -- bash -c 'while true; do sleep 1; done'
Launched the plugin getintocontainer-1642621594-1642621594 successfully 
You may check the log: pluginctl logs getintocontainer-1642621594-1642621594
To terminate the job: pluginctl rm getintocontainer-1642621594-1642621594
```

We use `deploy` to run a container named "getintocontainer" with the arguments that has an infinite loop to sleep. This sleep is needed for us to buy time while we are executing commands inside the container. Without this sleep mechanism, the container would execute default command which does nothing and terminate almost immediately.

```bash
$ pluginctl ps
NAME                                     STATUS      START_TIME                RUNNING_TIME
getintocontainer-1642621594-1642621594   Running     2022/01/19 19:46:34 UTC   7.57783992s
```

We run `ps` subcommand to check if the container is running. If it is not running state, the container might have been downloading the image from Internet, if the system does not have the image cached.

```bash
$ pluginctl exec -ti getintocontainer-1642621594-1642621594 -- /bin/bash
root@getintocontainer-1642621594-1642621594-nfrcj:/# date
Wed Jan 19 19:51:05 UTC 2022
root@getintocontainer-1642621594-1642621594-nfrcj:/# echo "I am inside container"
I am inside container
root@getintocontainer-1642621594-1642621594-nfrcj:/# exit
exit
```

Now we run `exec` subcommand with `-ti` options indicating that we attach your terminal to the shell inside the container. This allows you to use your terminal to execute commands inside the container. We then execute date and echo commands there, see the result, and exit. This exit exits the shell and detaches the terminal from the shell.

```bash
pluginctl rm getintocontainer-1642621594-1642621594
Terminated the plugin getintocontainer-1642621594-1642621594 successfully
```

We have one last thing, clean up. Because the container would run forever due to the infinite loop and sleep, we need to terminate the container. The `rm` subcommand deletes the container from the system.