#!/bin/bash

sesctl_path=./out/sesctl
dir_path=$1
find ${dir_path} -type f -exec ${sesctl_path} submit -f {} \;
