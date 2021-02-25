#!/bin/bash

if [ $# -eq 0 ]; then
  echo "Usage: run_ns.sh NODE_ID CS_URI RMQ_MGNT_URI"
  echo "       ex) run_ns.sh 000000000001 http://10.31.81.1:9770 http://10.31.81.10:15672"
  exit 1
fi

nodeid=$1
cloudscheduler_uri=$2
rmq_mgnt_uri=$3
docker run -ti --rm \
  --name ns \
  --entrypoint /app/nodescheduler \
  --network host \
  -v /etc/rancher/k3s/k3s.yaml:/app/k3s.yaml:ro \ 
  gemblerz/scheduler:1.0.0 \
  -kubeconfig /app/k3s.yaml \
  -registry gemblerz/ \
  -nodeid ${nodeid} \
  -cloudscheduler-uri ${cloudscheduler_uri} \
  -rabbitmq-management-uri ${rmq_mgnt_uri}

