#!/bin/bash

filepath=$1
CS_API="http://localhost:9770/api/v1"
curl -X PUT -T ${filepath} ${CS_API}/submit
