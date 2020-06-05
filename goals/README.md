### Node Configuration Generation


#### Testing The Configuration Generation Process

A fake edge code repository (ECR) supports the process as the process needs to access the ECR to retreive information about the registered applications. The fake ECR hosts a local webserver via Flask and accepts requests on registering apps, returning information about queried app, and listing the apps it has.

To register a set of fake apps,
```
$ python3 fake_ecr.py
# on a new terminal
$ python3 insert_ecr.py fake_apps.yaml
# to test if the fake ecr has registered the fake apps
$ curl http://localhost:5000/list
{'id': 'c76e79c8-e59c-43d7-9847-2cd6f2904af7', 'name': 'rabbitmq', 'version': '3.6-node'}<br>{'id': '8663add1-9075-4920-8ce5-4ad33d6bff75', 'name': 'plugin-metsense', 'version': '4.1.1'}<br>{'id': 'c63b4146-7fd2-4013-8375-c67865d74284', 'name': 'plugin-media-streaming', 'version': '0.1.0'}<br>{'id': 'fee79087-9d56-4449-8c68-9c3e0fe404b6', 'name': 'water-analyzer', 'version': '0.1.0'}<br>
```

