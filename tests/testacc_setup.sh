#!/bin/bash

source "$(pwd)"/tests/env.sh
docker-compose -f "$(pwd)"/tests/docker-compose.yaml up -d
"$(pwd)"/tests/wait-cassandra-docker.sh "$(pwd)"/tests/docker-compose.yaml
