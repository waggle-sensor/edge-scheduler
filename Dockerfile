FROM golang:1.20 as builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION
ENV VERSION=${VERSION}
RUN go build -ldflags "-X main.Version=${VERSION}" -o ./ ./...

FROM python:3.11
WORKDIR /app

COPY requirements.txt .
RUN pip3 install -r requirements.txt

COPY pkg/knowledgebase/kb.py pkg/knowledgebase/util /app/

COPY --from=builder /build/cloudscheduler /usr/bin/cloudscheduler
COPY --from=builder /build/nodescheduler /usr/bin/nodescheduler
COPY --from=builder /build/pluginctl /usr/bin/pluginctl
COPY --from=builder /build/runplugin /usr/bin/runplugin
