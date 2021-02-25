#!/bin/bash

echo "WARNING: this MUST be called from the project root directory"
image=gemblerz/scheduler:1.0.0

docker rm -f cs

docker run -d \
  --entrypoint /app/cloudscheduler \
  -p 9770:9770 \
  -v $(pwd)/data/:/app/data/ \
  --name cs \
  ${image}
