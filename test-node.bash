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

if [[ $# -gt 0 ]] && [[ $1 == "script" ]]; then
    shift
    docker-compose run testnode-scripts "$@"
    exit $?
fi

num_volumes=`docker volume ls --filter label=com.docker.compose.project=nitro -q | wc -l`

if [[ $num_volumes -eq 0 ]]; then
    force_init=true
else
    force_init=false
fi

run=true
force_build=false
validate=false
detach=false
blockscout=true
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
        --no-blockscout)
            blockscout=false
            shift
            ;;
        --no-run)
            run=false
            shift
            ;;
        --detach)
            detach=true
            shift
            ;;
        *)
            echo Usage: $0 \[OPTIONS..]
            echo        $0 script [SCRIPT-ARGS]
            echo
            echo OPTIONS:
            echo --build:           rebuild docker image
            echo --init:            remove all data, rebuild, deploy new rollup
            echo --validate:        heavy computation, validating all blocks in WASM
            echo --detach:          detach from nodes after running them
            echo --no-run:          does not launch nodes \(usefull with build or init\)
            echo
            echo script rus inside a separate docker. For SCRIPT-ARGS, run $0 script --help
            exit 0
    esac
done

if $force_init; then
    force_build=true
fi

NODES="sequencer"
if $validate; then
    NODES="$NODES validator"
else
    NODES="$NODES staker-unsafe"
fi
if $blockscout; then
    NODES="$NODES blockscout"
fi

if $force_build; then
    echo == Building..
    docker-compose build --no-rm $NODES testnode-scripts
fi

if $force_init; then
    echo == Removing old data..
    docker-compose down
    leftoverContainers=`docker container ls -a --filter label=com.docker.compose.project=nitro -q | xargs echo`
    if [ `echo $leftoverContainers | wc -w` -gt 0 ]; then
        docker rm $leftoverContainers
    fi
    docker volume prune -f --filter label=com.docker.compose.project=nitro

    echo == Generating l1 keys
    docker-compose run --entrypoint sh geth -c "echo passphrase > /root/.ethereum/passphrase"
    docker-compose run --entrypoint sh geth -c "echo e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2 > /root/.ethereum/tmp-funnelkey"
    docker-compose run geth account import --password /root/.ethereum/passphrase --keystore /keystore /root/.ethereum/tmp-funnelkey
    docker-compose run --entrypoint sh geth -c "rm /root/.ethereum/tmp-funnelkey"
    docker-compose run geth account new --password /root/.ethereum/passphrase --keystore /keystore 
    docker-compose run geth account new --password /root/.ethereum/passphrase --keystore /keystore 

    echo == funding validator and sequencer, writing configs
    docker-compose run testnode-scripts --l1fund --ethamount 1000 --l1account validator
    docker-compose run testnode-scripts --l1fund --ethamount 1000 --l1account sequencer
    docker-compose run testnode-scripts --writeconfig

    echo == Deploying L2
    validaotraddress=`docker-compose run testnode-scripts --l1account sequencer --printaddress | tail -n 1 | tr -d '\r\n'`
    docker-compose run --entrypoint target/bin/deploy sequencer -l1conn ws://geth:8546 -l1keystore /l1keystore -l1DeployAccount $validaotraddress -l1deployment /config/deployment.json -authorizevalidators 10

    docker-compose run testnode-scripts --bridgefunds --ethamount 100000
fi

if $run; then
    UP_FLAG=""
    if $detach; then
        UP_FLAG="-d"
    fi

    echo == Launching Sequencer
    echo if things go wrong - use --init to create a new chain
    echo

    docker-compose up $UP_FLAG $NODES
fi
