#!/usr/bin/env bash

set -e

NITRO_NODE_VERSION=offchainlabs/nitro-node:v2.0.10-73224e3-dev
BLOCKSCOUT_VERSION=offchainlabs/blockscout:v1.0.0-c8db5b1

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
tokenbridge=true
consensusclient=false
redundantsequencers=0
dev_build_nitro=false
dev_build_blockscout=false
batchposters=1
devprivkey=b6b15c8cb491557369f3c7d2c287b053eb229daa9c22138887752191c9520659
l1chainid=1337
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
        --dev)
            dev_build_nitro=true
            dev_build_blockscout=true
            shift
            while [[ $# -gt 0 && $1 != -* ]]; do
                if [[ $1 == "nitro" ]]; then
                    dev_build_nitro=true
                elif [[ $1 == "blockscout" ]]; then
                    dev_build_blockscout=true
                fi
                shift
            done
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
        --no-tokenbridge)
            tokenbridge=false
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
        --batchposters)
            batchposters=$2
            if ! [[ $batchposters =~ [0-3] ]] ; then
                echo "batchposters must be between 0 and 3 value:$batchposters."
                exit 1
            fi
            shift
            shift
            ;;
        --pos)
            consensusclient=true
            l1chainid=32382
            shift
            ;;
        --redundantsequencers)
            redundantsequencers=$2
            if ! [[ $redundantsequencers =~ [0-3] ]] ; then
                echo "redundantsequencers must be between 0 and 3 value:$redundantsequencers."
                exit 1
            fi
            shift
            shift
            ;;
        *)
            echo Usage: $0 \[OPTIONS..]
            echo        $0 script [SCRIPT-ARGS]
            echo
            echo OPTIONS:
            echo --build:           rebuild docker images
            echo --dev:             build nitro and blockscout dockers from source \(otherwise - pull docker\)
            echo --init:            remove all data, rebuild, deploy new rollup
            echo --pos:             l1 is a proof-of-stake chain \(using prism for consensus\)
            echo --validate:        heavy computation, validating all blocks in WASM
            echo --batchposters:    batch posters [0-3]
            echo --redundantsequencers redundant sequencers [0-3]
            echo --detach:          detach from nodes after running them
            echo --no-blockscout:   don\'t build or launch blockscout
            echo --no-tokenbridge:  don\'t build or launch tokenbridge
            echo --no-run:          does not launch nodes \(usefull with build or init\)
            echo
            echo script rus inside a separate docker. For SCRIPT-ARGS, run $0 script --help
            exit 0
    esac
done

if $force_init; then
  force_build=true
fi

if $dev_build_nitro; then
  if [[ "$(docker images -q nitro-node-dev:latest 2> /dev/null)" == "" ]]; then
    force_build=true
  fi
fi

if $dev_build_blockscout; then
  if [[ "$(docker images -q blockscout:latest 2> /dev/null)" == "" ]]; then
    force_build=true
  fi
fi

NODES="sequencer"
INITIAL_SEQ_NODES="sequencer"

if [ $redundantsequencers -gt 0 ]; then
    NODES="$NODES sequencer_b"
    INITIAL_SEQ_NODES="$INITIAL_SEQ_NODES sequencer_b"
fi
if [ $redundantsequencers -gt 1 ]; then
    NODES="$NODES sequencer_c"
fi
if [ $redundantsequencers -gt 2 ]; then
    NODES="$NODES sequencer_d"
fi

if [ $batchposters -gt 0 ]; then
    NODES="$NODES poster"
fi
if [ $batchposters -gt 1 ]; then
    NODES="$NODES poster_b"
fi
if [ $batchposters -gt 2 ]; then
    NODES="$NODES poster_c"
fi


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
  if $dev_build_nitro; then
    docker build . -t nitro-node-dev --target nitro-node-dev
  fi
  if $dev_build_blockscout; then
    if $blockscout; then
      docker build blockscout -t blockscout -f blockscout/docker/Dockerfile
    fi
  fi
  LOCAL_BUILD_NODES=testnode-scripts
  if $tokenbridge; then
    LOCAL_BUILD_NODES="$LOCAL_BUILD_NODES testnode-tokenbridge"
  fi
  docker-compose build --no-rm $LOCAL_BUILD_NODES
fi

if $dev_build_nitro; then
  docker tag nitro-node-dev:latest nitro-node-dev-testnode
else
  docker pull $NITRO_NODE_VERSION
  docker tag $NITRO_NODE_VERSION nitro-node-dev-testnode
fi

if $dev_build_blockscout; then
  if $blockscout; then
    docker tag blockscout:latest blockscout-testnode
  fi
else
  if $blockscout; then
    docker pull $BLOCKSCOUT_VERSION
    docker tag $BLOCKSCOUT_VERSION blockscout-testnode
  fi
fi

if $force_build; then
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
    docker-compose run testnode-scripts write-accounts
    docker-compose run --entrypoint sh geth -c "echo passphrase > /datadir/passphrase"
    docker-compose run --entrypoint sh geth -c "chown -R 1000:1000 /keystore"
    docker-compose run --entrypoint sh geth -c "chown -R 1000:1000 /config"

    if $consensusclient; then
      echo == Writing configs
      docker-compose run testnode-scripts write-geth-genesis-config

      echo == Writing configs
      docker-compose run testnode-scripts write-prysm-config

      echo == Initializing go-ethereum genesis configuration
      docker-compose run geth init --datadir /datadir/ /config/geth_genesis.json

      echo == Starting geth
      docker-compose up -d geth

      echo == Creating prysm genesis
      docker-compose up create_beacon_chain_genesis

      echo == Running prysm
      docker-compose up -d prysm_beacon_chain
      docker-compose up -d prysm_validator
    else
      docker-compose up -d geth
    fi

    echo == Funding validator and sequencer
    docker-compose run testnode-scripts send-l1 --ethamount 1000 --to validator --wait
    docker-compose run testnode-scripts send-l1 --ethamount 1000 --to sequencer --wait

    echo == create l1 traffic
    docker-compose run testnode-scripts send-l1 --ethamount 1000 --to user_l1user --wait
    docker-compose run testnode-scripts send-l1 --ethamount 0.0001 --from user_l1user --to user_l1user_b --wait --delay 500 --times 500 > /dev/null &


    echo == Deploying L2
    sequenceraddress=`docker-compose run testnode-scripts print-address --account sequencer | tail -n 1 | tr -d '\r\n'`

    docker-compose run --entrypoint /usr/local/bin/deploy poster --l1conn ws://geth:8546 --l1keystore /home/user/l1keystore --sequencerAddress $sequenceraddress --ownerAddress $sequenceraddress --l1DeployAccount $sequenceraddress --l1deployment /config/deployment.json --authorizevalidators 10 --wasmrootpath /home/user/target/machines --l1chainid=$l1chainid

    echo == Writing configs
    docker-compose run testnode-scripts write-config

    echo == Initializing redis
    docker-compose run testnode-scripts redis-init --redundancy $redundantsequencers

    echo == Funding l2 funnel
    docker-compose up -d $INITIAL_SEQ_NODES
    docker-compose run testnode-scripts bridge-funds --ethamount 100000 --wait

    if $tokenbridge; then
        echo == Deploying token bridge
        docker-compose run -e ARB_KEY=$devprivkey -e ETH_KEY=$devprivkey testnode-tokenbridge gen:network
        docker-compose run --entrypoint sh testnode-tokenbridge -c "cat localNetwork.json"
        echo
    fi
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
