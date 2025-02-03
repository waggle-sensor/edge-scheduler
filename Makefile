VERSION?=0.0.0

cli-all-arch: cli-linux-amd64 cli-linux-arm64 cli-darwin-amd64 cli-darwin-arm64 cli-windows-amd64
	cd out && shasum -a 1 runplugin* pluginctl* sesctl* > sha1sum.txt
	cd out && shasum -a 256 runplugin* pluginctl* sesctl* > sha256sum.txt

cli-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/runplugin-linux-arm64 ./cmd/runplugin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/pluginctl-linux-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/sesctl-linux-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./out/runplugin-linux-amd64 ./cmd/runplugin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./out/pluginctl-linux-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./out/sesctl-linux-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./out/runplugin-darwin-amd64 ./cmd/runplugin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./out/pluginctl-darwin-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./out/sesctl-darwin-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o ./out/runplugin-darwin-arm64 ./cmd/runplugin
	GOOS=darwin GOARCH=arm64 go build -o ./out/pluginctl-darwin-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	GOOS=darwin GOARCH=arm64 go build -o ./out/sesctl-darwin-arm64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./out/runplugin-windows-amd64 ./cmd/runplugin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./out/pluginctl-windows-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./out/sesctl-windows-amd64 -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

cli:
	CGO_ENABLED=0 go build -o ./out/pluginctl -ldflags "-X main.Version=${VERSION}" ./cmd/pluginctl
	CGO_ENABLED=0 go build -o ./out/sesctl -ldflags "-X main.Version=${VERSION}" ./cmd/sesctl

scheduler-all-arch: scheduler-amd64 scheduler-arm64

scheduler-amd64:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./out/cloudscheduler-amd64 -ldflags "-X main.Version=${VERSION}" cmd/cloudscheduler/main.go
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ./out/nodescheduler-amd64 -ldflags "-X main.Version=${VERSION}" cmd/nodescheduler/main.go

scheduler-arm64:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o ./out/cloudscheduler-arm64 -ldflags "-X main.Version=${VERSION}" cmd/cloudscheduler/main.go
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o ./out/nodescheduler-arm64 -ldflags "-X main.Version=${VERSION}" cmd/nodescheduler/main.go

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
