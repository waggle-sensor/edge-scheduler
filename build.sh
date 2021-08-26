#!/bin/bash
GOOS=linux GOARCH=amd64 go build -o ./runplugin-amd64 ./cmd/runplugin
GOOS=linux GOARCH=arm64 go build -o ./runplugin-arm64 ./cmd/runplugin
