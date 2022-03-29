#!/bin/bash

set -eu

PLUGINCTL_VERSION="${PLUGINCTL_VERSION:=0.0.0}"

GOOS=linux GOARCH=amd64 go build -o ./runplugin-amd64 ./cmd/runplugin
GOOS=linux GOARCH=arm64 go build -o ./runplugin-arm64 ./cmd/runplugin

GOOS=linux GOARCH=amd64 go build -o ./pluginctl-amd64 -ldflags "-X main.Version=${PLUGINCTL_VERSION}" ./cmd/pluginctl
GOOS=linux GOARCH=arm64 go build -o ./pluginctl-arm64 -ldflags "-X main.Version=${PLUGINCTL_VERSION}" ./cmd/pluginctl
