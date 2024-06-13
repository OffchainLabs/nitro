#!/bin/bash

anvil --fork-url $L1_RPC > /dev/null &

anvil_pid=$!

yarn script:bold-prepare && \
yarn script:bold-populate-lookup && \
yarn script:bold-local-execute

kill $anvil_pid
