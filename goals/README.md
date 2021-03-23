### Node Configuration Generation

One of the SES outputs is a node configuration that is later pulled by target nodes and interpreted by the local scheduler running inside the target nodes. The node configuration is a guide book given from the cloud that helps nodes determine what to do in order to support the scientific problem. The guide book provides many sections and troubleshootings that each specifies what to do when something occurrs. This allows the local scheduler to switch between sections depending on the troubleshooting guide and the context perceived locally.

#### Emulating The Configuration Generation Process

A fake edge code repository (ECR) supports the process of node configuration generation as the process needs to access ECR to retreive information about the registered applications. The fake ECR hosts a local webserver via Flask and accepts requests on registering apps, returning information about queried app, and listing the apps that have been registered.

To register a set of fake apps,
```
$ python3 fake_ecr.py
# on a new terminal
$ python3 insert_ecr.py fake_apps.yaml
# to test if the fake ecr has registered the fake apps
$ curl http://localhost:5000/listapp
{'id': 'c76e79c8-e59c-43d7-9847-2cd6f2904af7', 'name': 'rabbitmq', 'version': '3.6-node'}<br>{'id': '8663add1-9075-4920-8ce5-4ad33d6bff75', 'name': 'plugin-metsense', 'version': '4.1.1'}<br>{'id': 'c63b4146-7fd2-4013-8375-c67865d74284', 'name': 'plugin-media-streaming', 'version': '0.1.0'}<br>{'id': 'fee79087-9d56-4449-8c68-9c3e0fe404b6', 'name': 'water-analyzer', 'version': '0.1.0'}<br>
```

Then run the script to generate a goal,
```
$ python3 translate.py -f job1_example.json
body:
  app_config:
  - default:
    - conditions:
      - 'True'
    - spec:
        containers:
        - image: waggle/rabbitmq:3.6-node
          name: rabbitmq
          ports:
          - containerPort: 15672
          resources:
            limits:
            - cpu: 1500m
            - memory: 256Mi
  rules: []
  sensor_config:
    spec:
      containers: []
header:
  goal_id: 32fe2840-c385-4ef1-8746-3d57767e4b44
  goal_name: simple job
  priority: 50
  target_nodes:
  - 001e06107e48
  user_id: gemblerz
```
