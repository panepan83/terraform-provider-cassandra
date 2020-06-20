#!/bin/bash

source "$(pwd)"/tests/env.sh
docker-compose -f "$(pwd)"/tests/docker-compose.yaml down
unset TF_ACC
