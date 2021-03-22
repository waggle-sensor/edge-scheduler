build:
	echo "Building cloudscheduler..."
	go build -o bin/cloudscheduler cmd/cloudscheduler/main.go
	echo "Building nodescheduler..."
	go build -o bin/nodescheduler cmd/nodescheduler/main.go
	cp pkg/knowledgebase/*.py bin/
	cp -r pkg/knowledgebase/util bin/util

docker:
	docker build -t gemblerz/scheduler:1.0.0 .