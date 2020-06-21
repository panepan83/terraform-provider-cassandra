#!/bin/bash

source "$(pwd)"/tests/env.sh
docker-compose -f "$(pwd)"/tests/docker-compose.yaml down
unset TF_ACC CASSANDRA_USERNAME CASSANDRA_PASSWORD CASSANDRA_PORT CASSANDRA_HOST
