FROM golang:1.15.3-alpine3.12

RUN apk --no-cache update \
  && apk add \
  build-base \
  git \
  python3-dev \
  py3-pip \
  zeromq-dev \
  czmq-dev
# libczmq-dev

COPY requirements.txt /app/
RUN pip3 install -r /app/requirements.txt
