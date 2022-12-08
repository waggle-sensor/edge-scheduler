# Tutorial: Building A Plugin
This tutorial demonstrates how plugin developer can download their AI code into a Waggle node and build a plugin container inside the node to test the code.

First, we download our example code (plugin-hello-world) to a Waggle node. If you have a code ready for running you can download yours instead.

__NOTE: This demonstration happens inside a Waggle node. Yes, we do want to git clone on the Waggle node__
```bash
git clone https://github.com/waggle-sensor/plugin-hello-world.git
```

The pluginctl's `build` subcommand utilizes Docker engine as a backend to build a container. The plugin-hello-world repository has a Dockerfile. To see the Dockerfile,
```bash
cat plugin-hello-world/Dockerfile
```

Content of the Dockerfile is,
```
# A Dockerfile is used to define how your code will be packaged. This includes
# your code, the base image and any additional dependencies you need.
FROM waggle/plugin-base:1.1.1-base

WORKDIR /app

# Now we include the Python requirements.txt file and install any missing dependencies.
COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

# Next, we add our code.
COPY main.py .

# Finally, we specify the "main" thing that should be run.
ENTRYPOINT [ "python3", "main.py" ]
```

Let's build the plugin,
```bash
pluginctl build plugin-hello-world
```

The command will output logs of the build,
```bash
Sending build context to Docker daemon  1.023MB
Step 1/6 : FROM waggle/plugin-base:1.1.1-base
 ---> 306bc3c3b60b
Step 2/6 : WORKDIR /app
...
Successfully built f5da6a99cd5e
Successfully tagged 10.31.81.1:5000/local/plugin-hello-world:latest
The push refers to repository [10.31.81.1:5000/local/plugin-hello-world]
fb7817cfd02b: Pushed
...
Successfully built plugin

10.31.81.1:5000/local/plugin-hello-world
```

In the end, it outputs the full path of the plugin container image for you to use. To run the plugin on the node,
```bash
pluginctl run --name builtplugin --selector node-role.kubernetes.io/master=true 10.31.81.1:5000/local/plugin-hello-world
```

The `run` subcommand will run the plugin and output "publishing hello world!",
```bash
Scheduled the plugin builtplugin successfully 
INFO: 2022/12/06 16:31:06 run.go:75: Plugin is in "Pending" state. Waiting...
INFO: 2022/12/06 16:31:08 run.go:75: Plugin is in "Pending" state. Waiting...
publishing hello world!
```

As a reminder, this is only for plugin development. The built plugin container is cached on the Waggle node and may not be run on other nodes unless we build it on the other nodes. If you are ready to make the code a plugin, please follow [this tutorial](https://docs.waggle-edge.ai/docs/tutorials/edge-apps/publishing-to-ecr).