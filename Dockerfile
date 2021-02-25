FROM golang:1.15.3-alpine3.12 as base

RUN apk --no-cache update \
  && apk add \
  build-base \
  git \
  python3-dev \
  py3-pip \
  zeromq-dev \
  czmq-dev
# libczmq-dev

FROM base as builder
WORKDIR $GOPATH/src/github.com/sagecontinuum/ses
COPY . .
RUN go build -o /app/cloudscheduler cmd/cloudscheduler/main.go \
  && go build -o /app/nodescheduler cmd/nodescheduler/main.go \
  && cp pkg/knowledgebase/kb.py /app/ \
  && cp -r pkg/knowledgebase/util /app/

FROM base
COPY --from=builder /app/ /app/

WORKDIR /app
COPY requirements.txt /app/
RUN pip3 install -r /app/requirements.txt
