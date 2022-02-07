#! /bin/bash

nodeID=$1

if [ -z ${nodeID} ]; then
  echo "Please specify node name (or ID)."
  echo "Usage: ./configure_nodescheduler.sh 00000000000001"
  exit 1
fi

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
      priorityClassName: wes-high-priority
      serviceAccountName: wes-plugin-scheduler
      containers:
      # TODO change to the real scheduler image :)
      - image: waggle/scheduler:0.9.2
        name: wes-plugin-scheduler
        command: ["/app/nodescheduler"]
        args:
        - -in-cluster
        envFrom:
        - configMapRef:
          name: wes-identity
        env:
        - name: WAGGLE_APP_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.uid
        WAGGLE_APP_ID
        resources:
          limits:
            cpu: 200m
            memory: 150Mi
          requests:
            cpu: 100m
            memory: 100Mi
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: wes-plugin-scheduler-view
  namespace: default
roleRef:
  kind: ClusterRole
  name: edit
  apiGroup: rbac.authorization.k8s.io
  # `edit` is a built-in cluster role. more info about these can be found here:
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
