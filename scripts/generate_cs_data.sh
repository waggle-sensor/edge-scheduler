#!/bin/bash

mkdir nodes
python3 create_nodes.py \
  -node-url https://api.sagecontinuum.org/production \
  -output-path ./nodes/

mkdir plugins
python3 create_plugins.py \
  -plugin-url https://ecr.sagecontinuum.org/api/apps \
  -output-path ./plugins/