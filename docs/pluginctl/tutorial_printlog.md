# Printing Logs
This tutorial demonstrates how to print logs out from a running container. This will be useful to check if your container is doing what it is supposed to do with sensors and inputs in our system.

```bash
$ pluginctl deploy --name printlog waggle/plugin-base:1.1.1-base -- bash -c 'while true; do date; sleep 1; done'
Launched the plugin printlog-1642626747-1642626747 successfully 
You may check the log: pluginctl logs printlog-1642626747-1642626747
To terminate the job: pluginctl rm printlog-1642626747-1642626747
```

We again use `deploy` subcommand to run a container that this time prints current time with 1 second interval. The print is being logged inside the container.

```bash
$ pluginctl logs printlog-1642626747-1642626747
Wed Jan 19 21:12:28 UTC 2022
Wed Jan 19 21:12:29 UTC 2022
Wed Jan 19 21:12:30 UTC 2022
```

The `logs` subcommand brings log messages from the container to the terminal.

```bash
$ pluginctl logs -f printlog-1642626747-1642626747
Wed Jan 19 21:12:28 UTC 2022
Wed Jan 19 21:12:29 UTC 2022
Wed Jan 19 21:12:30 UTC 2022
Wed Jan 19 21:12:31 UTC 2022
Wed Jan 19 21:12:32 UTC 2022
Wed Jan 19 21:12:33 UTC 2022
Wed Jan 19 21:12:34 UTC 2022
Wed Jan 19 21:12:35 UTC 2022
Wed Jan 19 21:12:36 UTC 2022
Wed Jan 19 21:12:37 UTC 2022
```

The `-f` -- follow -- option makes the log keep printed until interrupted (e.g., by ctrl + c).

```bash
$ pluginctl rm printlog-1642626747-1642626747
Terminated the plugin printlog-1642626747-1642626747 successfully
```

Finally we clean up the container as it would be running indefinitely.