#! /bin/bash

RMQ_NAME="svc/wes-rabbitmq"

rmqctl() {
	# docker exec ${RMQ_NAME} rabbitmqctl "$@"
	kubectl exec ${RMQ_NAME} -- rabbitmqctl "$@"
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



