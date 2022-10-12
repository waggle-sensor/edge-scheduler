VERSION?=0.0.0
cli-all-arch: cli-amd64 cli-arm64

cli-arm64:
	GOOS=linux GOARCH=arm64 go build -o ./out/runplugin-arm64 ./cmd/runplugin
	GOOS=linux GOARCH=arm64 go build -o ./out/pluginctl-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	GOOS=linux GOARCH=arm64 go build -o ./out/sesctl-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli-amd64:
	GOOS=linux GOARCH=amd64 go build -o ./out/runplugin-amd64 ./cmd/runplugin
	GOOS=linux GOARCH=amd64 go build -o ./out/pluginctl-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	GOOS=linux GOARCH=amd64 go build -o ./out/sesctl-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli:
	go build -o ./out/pluginctl -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	go build -o ./out/sesctl -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

scheduler-all-arch: scheduler-amd64 scheduler-arm64

scheduler-amd64:
	GOOS=linux GOARCH=amd64 go build -o ./out/cloudscheduler-amd64 -ldflags "-X main.Version=${VERSION}" cmd/cloudscheduler/main.go
	GOOS=linux GOARCH=amd64 go build -o ./out/nodescheduler-amd64 -ldflags "-X main.Version=${VERSION}" cmd/nodescheduler/main.go

scheduler-arm64:
	GOOS=linux GOARCH=arm64 go build -o ./out/cloudscheduler-arm64 -ldflags "-X main.Version=${VERSION}" cmd/cloudscheduler/main.go
	GOOS=linux GOARCH=arm64 go build -o ./out/nodescheduler-arm64 -ldflags "-X main.Version=${VERSION}" cmd/nodescheduler/main.go

scheduler:
	go build -o ./out/cloudscheduler -ldflags "-X main.Version=${VERSION}" ./cmd/cloudscheduler
	go build -o ./out/nodescheduler -ldflags "-X main.Version=${VERSION}" ./cmd/nodescheduler

docker:
	docker buildx build -t waggle/scheduler:${VERSION} --build-arg VERSION=${VERSION} --platform linux/amd64,linux/arm64 --push .

docker-pre-arm64:
	docker build -t waggle/scheduler:${VERSION}-pre --build-arg VERSION=${VERSION}-pre --build-arg TARGETARCH=arm64 .
	docker push waggle/scheduler:${VERSION}-pre

docker-pre-amd64:
	docker build -t waggle/scheduler:${VERSION}-pre --build-arg VERSION=${VERSION}-pre --build-arg TARGETARCH=amd64 .
	docker push waggle/scheduler:${VERSION}-pre
