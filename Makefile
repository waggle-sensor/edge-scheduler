version?=0.0.0

build:
	echo "Building cloudscheduler..."
	go build -o bin/cloudscheduler cmd/cloudscheduler/main.go
	echo "Building nodescheduler..."
	go build -o bin/nodescheduler cmd/nodescheduler/main.go
	cp pkg/knowledgebase/*.py bin/
	cp -r pkg/knowledgebase/util bin/util

cli:
	go build -o bin/sesctl cmd/cli/main.go

docker:
	docker build -t waggle/scheduler:${version} .
