# Printing Logs
This tutorial demonstrates how to print out logs from a running plugin. This will be useful to check if your container is doing what it is supposed to do with sensors and inputs in our system.

We first deploy a plugin that prints current date inside the container,
```bash
$ pluginctl deploy --name printlog waggle/plugin-base:1.1.1-base -- bash -c 'while true; do date; sleep 1; done'
```
The output would look like,
```bash
Launched the plugin printlog successfully 
You may check the log: pluginctl logs printlog
To terminate the job: pluginctl rm printlog
```

We use `logs` subcommand to access to logs of a plugin. It brings the log to user terminal,
```bash
$ pluginctl logs printlog
```

Dates are printed with 1 second interval,
```bash
Wed Jan 19 21:12:28 UTC 2022
Wed Jan 19 21:12:29 UTC 2022
Wed Jan 19 21:12:30 UTC 2022
```

The `logs` subcommand can follow logs using `-f` option. This will be useful to keep monitoring what the plugin outputs,
```bash
$ pluginctl logs -f printlog
```

Again, the output would be printed dates,
```bash
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

You can interrupt the terminal (e.g., by ctrl + c) to close the logging. Or, it can end when the plugin terminates, however in this tutorial it is not possible because the plugin runs forever.

Finally, we clean up the plugin as it would be running indefinitely.
```bash
$ pluginctl rm printlog
```

The output would look like,
```bash
Terminated the plugin printlog successfully
```
