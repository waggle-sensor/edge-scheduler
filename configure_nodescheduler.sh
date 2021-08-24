#! /bin/bash

RMQ_NAME="wes-rabbitmq"
RMQ_NAME="beehive-rabbitmq"
nodeID=$1

if [ -z ${nodeID} ]; then
  echo "Please specify node name (or ID)."
  echo "Usage: ./configure_nodescheduler.sh 00000000000001"
  exit 1
fi

rmqctl() {
	docker exec ${RMQ_NAME} rabbitmqctl "$@"
	# kubectl exec svc/${RMQ_NAME} -- rabbitmqctl "$@"
}

username="wes-plugin-scheduler"
password="$(openssl rand -hex 20)"
confperm=".*"
writeperm=".*"
readperm=".*"

echo "Generating a RabbitMQ account for node scheduler..."
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

echo "Generating k3s objects for node scheduler..."
cat <<EOF > nodescheduler.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wes-plugin-scheduler
spec:
  selector:
    matchLabels:
      app: wes-plugin-scheduler
  template:
    metadata:
      labels:
        app: wes-plugin-scheduler
    spec:
      serviceAccountName: wes-plugin-scheduler
      containers:
      - image: waggle/scheduler:0.4.0
        name: wes-plugin-scheduler
        command: ["/app/nodescheduler"]
        args:
        - "-nodeid"
        - "${nodeID}"
        - "-rabbitmq-uri"
        - "${RMQ_NAME}:5672"
        - "-rabbitmq-username"
        - "${username}"
        - "-rabbitmq-password"
        - "${password}"
        - "-rabbitmq-management-uri"
        - "http://${RMQ_NAME}:15672"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: wes-plugin-scheduler-view
  namespace: default
roleRef:
  kind: ClusterRole
  name: edit
  apiGroup: rbac.authorization.k8s.io
  # 'edit' is a built-in cluster role. more info about these can be found here:
  # https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles
subjects:
- kind: ServiceAccount
  name: wes-plugin-scheduler
  namespace: default
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: wes-plugin-scheduler
EOF

echo "Done."
echo "To launch node scheduler, kubectl apply -f nodescheduler.yaml"