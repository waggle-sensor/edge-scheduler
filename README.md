# Sage Edge Scheduler (SES)

SES offers software components that allow users to submit "jobs" that run applications (user plugins) on edge computing nodes.

One cloud scheduler manages multiple node schedulers. Each node scheduler must contact to only one cloud scheduler. Users interact only with a cloud scheduler to submit and monitor jobs.

## Build Docker Image for SES

```
# Build a Docker image with the version 0.0.0
$ version=0.0.0 make docker
```

## How To Run Cloud/Node Schedulers

We assume that a Kubernetes computing cluster runs on each cloud and edge computing platform. Then, use [Cloud](kubernetes/cloudscheduler) Kubernetes objects to run the cloud scheduler in the cloud and use [Node](kubernetes/nodescheduler) Kubernetes objects to run node scheduler at the edge.

```
# Run cloud scheduler in the cloud computing cluster
$ kubectl apply -f kubernetes/cloudscheduler

# Run node scheduler in the edge computing cluster
$ kubectl apply -f kubernetes/nodescheduler
```
