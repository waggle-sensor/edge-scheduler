FROM waggle/plugin-base:1.1.1-base as base

RUN apt-get update \
  && apt-get install -y \
  build-essential \
  pkg-config \
  # build-base \
  wget \
  # libzmq3-dev \
#  zeromq-dev \
  # libczmq-dev \
#  czmq-dev \
  && rm -rf /var/lib/apt/lists/*
# libczmq-dev

ARG TARGETARCH
WORKDIR /tmp
RUN wget https://golang.org/dl/go1.19.1.linux-${TARGETARCH}.tar.gz \
  && rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.1.linux-${TARGETARCH}.tar.gz \
  && echo "PATH=\$PATH:/usr/local/go/bin" | tee -a $HOME/.bashrc \
  && rm go1.19.1.linux-${TARGETARCH}.tar.gz

FROM base as builder
WORKDIR $GOPATH/src/github.com/waggle-sensor/edge-scheduler
ARG TARGETARCH
COPY . .
ARG VERSION
ENV VERSION=${VERSION}
RUN export PATH=$PATH:/usr/local/go/bin:/usr/bin/pkg-config \
  && make scheduler-${TARGETARCH} cli-linux-${TARGETARCH} \
  && mkdir -p /app \
  && cp ./out/* /app/ \
  && cp pkg/knowledgebase/kb.py /app/ \
  && cp -r pkg/knowledgebase/util /app/

FROM base
ARG TARGETARCH
COPY requirements.txt /app/
RUN pip3 install -r /app/requirements.txt

COPY --from=builder /app/ /app/

RUN chmod +x /app/cloudscheduler-${TARGETARCH} \
  && ln -s /app/cloudscheduler-${TARGETARCH} /usr/bin/cloudscheduler \
  && chmod +x /app/nodescheduler-${TARGETARCH} \
  && ln -s /app/nodescheduler-${TARGETARCH} /usr/bin/nodescheduler \
  && chmod +x /app/pluginctl-linux-${TARGETARCH} \
  && ln -s /app/pluginctl-linux-${TARGETARCH} /usr/bin/pluginctl \
  && chmod +x /app/runplugin-linux-${TARGETARCH} \
  && ln -s /app/runplugin-linux-${TARGETARCH} /usr/bin/runplugin

WORKDIR /app
