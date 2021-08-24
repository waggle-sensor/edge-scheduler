#! /bin/bash

RMQ_NAME="beehive-rabbitmq"

rmqctl() {
	docker exec ${RMQ_NAME} rabbitmqctl "$@"
	# kubectl exec svc/wes-rabbitmq -- rabbitmqctl "$@"
}

username="cloudscheduler"
password="$(openssl rand -hex 20)"
confperm=".*"
writeperm=".*"
readperm=".*"

echo "Generating a RabbitMQ account for cloud scheduler..."
# from waggle-edge-stack/kubernetes/update-rabbitmq-auth.sh
# https://github.com/waggle-sensor/waggle-edge-stack/blob/main/kubernetes/update-rabbitmq-auth.sh
(
while ! rmqctl authenticate_user "$username" "$password"; do
    while ! (rmqctl add_user "$username" "$password" || rmqctl change_password "$username" "$password"); do
      sleep 3
    done
done

while ! rmqctl set_permissions "$username" "$confperm" "$writeperm" "$readperm"; do
  sleep 3
done
) &> /dev/null
echo "Done"

echo "Generating k3s objects for cloud scheduler..."
cat <<EOF > cloudscheduler.yaml
apiVersion: apps/v1 # this apiVersion is relevant as of Kubernetes 1.9
kind: Deployment
metadata:
  name: cloudscheduler
  labels:
    app: cloudscheduler
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloudscheduler
  template:
    metadata:
      labels:
        app: cloudscheduler
    spec:
      containers:
      - name: cloudscheduler
        image: waggle/scheduler:0.4.0
        volumeMounts:
        - mountPath: /app/data/
          name: data
        ports:
        - containerPort: 9770
        workingDir: /app/
        command: ["/app/cloudscheduler"]
        args:
        - "-rabbitmq-uri"
        - "${RMQ_NAME}"
        - "-rabbitmq-username"
        - "${username}"
        - "-rabbitmq-password"
        - "${password}"
        - "-data-dir"
        - "data"
        - "-port"
        - "9770"
      volumes:
      - name: data
        hostPath:
          path: /tmp/data
          type: Directory
---
apiVersion: v1
kind: Service
metadata:
  name: cloudscheduler
spec:
  selector:
    app: cloudscheduler
  ports:
  - name: api
    protocol: TCP
    port: 9770
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cloudscheduler-ingress
  annotations:
spec:
  rules:
  - host: ses.localhost
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: cloudscheduler
            port:
              number: 9770
EOF

echo "Done."
echo "To launch cloud scheduler, kubectl apply -f cloudscheduler.yaml"