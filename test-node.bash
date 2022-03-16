#!/bin/bash

set -e

mydir=`dirname $0`
cd "$mydir"

if ! which docker-compose > /dev/null; then
    echo == Error! docker-compose not installed
    echo please install docker-compose and have it in PATH
    echo see https://docs.docker.com/compose/install/
    exit 1
fi

num_volumes=`docker volume ls --filter label=com.docker.compose.project=nitro -q | wc -l`

if [[ $num_volumes -eq 0 ]]; then
    force_init=true
else
    force_init=false
fi

force_build=false
validate=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --init)
            if ! $force_init; then
                echo == Warning! this will remove all previous data
                read -p "are you sure? [y/n]" -n 1 response
                if [[ $response == "y" ]] || [[ $response == "Y" ]]; then
                    force_init=true
                    echo
                else
                    exit 0
                fi
            fi
            shift
            ;;
        --build)
            force_build=true
            shift
            ;;
        --validate)
            validate=true
            shift
            ;;
        *)
            echo Usage: $0 \[OPTIONS..]
            echo
            echo OPTIONS:
            echo --build:           rebuild docker image
            echo --init:            remove all data, rebuild, deploy new rollup
            echo --validate:        heavy computation, validating all blocks in WASM
            exit 0
    esac
done

if [[ $# -eq 1 ]]; then
    if [[ $1 == "--init" ]]; then
        if ! $force_init; then
            echo == Warning! this will remove all previous data
            read -p "are you sure? [y/n]" -n 1 response
            if [[ $response == "y" ]] || [[ $response == "Y" ]]; then
                force_init=true
                echo
            else
                exit 0
            fi
        fi
    elif [[ $1 == "--build" ]]; then
        force_build=true
    else
        echo Usage: $0 \[--init \| --build\]
        exit 0
    fi
fi

if $force_init; then
    force_build=true
fi

if $force_build; then
    echo == Building..
    docker-compose build --no-rm
fi

if $force_init; then
    echo == Removing old data..
    docker-compose down
    docker volume prune -f --filter label=com.docker.compose.project=nitro

    echo == Generating l1 key
    docker-compose run --entrypoint sh geth -c "echo passphrase > /root/.ethereum/passphrase"
    docker-compose run geth account new --password /root/.ethereum/passphrase --keystore /keystore

    echo == Deploying L2
    docker-compose run --entrypoint target/bin/deploy sequencer -l1conn ws://geth:8546 -l1keystore /l1keystore -l1deployment /deploydata/deployment.json -authorizevalidators 10
fi

echo == Launching Sequencer
echo if things go wrong - use --init to create a new chain
echo
if $validate; then
    STAKER_NODE="validator"
else
    STAKER_NODE="staker-unsafe"
fi
docker-compose up sequencer $STAKER_NODE
