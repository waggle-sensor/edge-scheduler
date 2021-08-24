# Sage Edge Scheduler (SES)

SES offers software components that allow users to submit "jobs" that run applications (user plugins) on edge computing nodes.

One cloud scheduler manages multiple node schedulers. Each node scheduler must contact to only one cloud scheduler. Users interact only with a cloud scheduler to submit and monitor jobs.

## Build Docker Image for SES

```
# Build a Docker image with the version 0.0.0
$ version=0.0.0 make docker
```

## Configure Cloud and Node Scheduler

[configure_cloudscheduler.sh](configure_cloudscheduler.sh) creates a RabbitMQ account for the cloud scheduler to push science goals to node schedulers and generates k3s objects to run the cloud scheduler in a Kubernetes cluster.

Waggle edge stack should already have [configured](https://github.com/waggle-sensor/waggle-edge-stack/blob/main/kubernetes/wes-plugin-scheduler.yaml) the cluster for the node scheduler

[configure_nodescheduler.sh](configure_nodecheduler.sh) creates a RabbitMQ account for the node scheduler to receive science goals from the cloud scheduler and generates k3s objects to run the node scheduler in a Kubernetes cluster.

## How To Run Cloud/Node Schedulers

We assume that a Kubernetes computing cluster runs on each cloud and edge computing platform. Then, use [Cloud](kubernetes/cloudscheduler) Kubernetes objects to run the cloud scheduler in the cloud and use [Node](kubernetes/nodescheduler) Kubernetes objects to run node scheduler at the edge.

```
# Run cloud scheduler in the cloud computing cluster
$ kubectl apply -f kubernetes/cloudscheduler

# Run node scheduler in the edge computing cluster
$ kubectl apply -f kubernetes/nodescheduler
```
