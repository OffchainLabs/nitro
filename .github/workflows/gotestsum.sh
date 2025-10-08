#!/bin/bash

check_missing_value() {
    if [[ $1 -eq 0 || $2 == -* ]]; then
        echo "missing $3 argument value"
        exit 1
    fi
}

timeout=""
tags=""
run=""
test_state_scheme=""
junitfile=""
log=true
race=false
cover=false
execution_consensus_jsonrpc_interconnect=false
flaky=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --timeout)
      shift
      check_missing_value $# "$1" "--timeout"
      timeout=$1
      shift
      ;;
    --tags)
      shift
      check_missing_value $# "$1" "--tags"
      tags=$1
      shift
      ;;
    --run)
      shift
      check_missing_value $# "$1" "--run"
      run=$1
      shift
      ;;
    --test_state_scheme)
      shift
      check_missing_value $# "$1" "--test_state_scheme"
      test_state_scheme=$1
      shift
      ;;
    --race)
      race=true
      shift
      ;;
    --cover)
      cover=true
      shift
      ;;
    --execution_consensus_jsonrpc_interconnect)
      execution_consensus_jsonrpc_interconnect=true
      shift
      ;;
    --nolog)
      log=false
      shift
      ;;
    --junitfile)
      shift
      check_missing_value $# "$1" "--junitfile"
      junitfile=$1
      shift
      ;;
    --flaky)
      flaky=true
      shift
      ;;
    *)
      echo "Invalid argument: $1"
      exit 1
      ;;
  esac
done

if [ "$flaky" == true ]; then
  if [ "$run" != "" ]; then
    run="Flaky/$run"
  else
    run="Flaky"
  fi
fi

# Add the gotestsum flags first
cmd="stdbuf -oL gotestsum --format short-verbose --packages=\"./...\" --rerun-fails=1 --rerun-fails-max-failures=30 --no-color=false"

if [ "$junitfile" != "" ]; then
  cmd="$cmd --junitfile \"$junitfile\""
fi

# Append the separator and go test arguments
cmd="$cmd --"

if [ "$timeout" != "" ]; then
  cmd="$cmd -timeout $timeout"
fi

if [ "$tags" != "" ]; then
  cmd="$cmd -tags=$tags"
fi

if [ "$run" != "" ]; then
  cmd="$cmd -run=$run"
fi

if [ "$flaky" == false ]; then
  cmd="$cmd -skip=Flaky"
fi

if [ "$race" == true ]; then
  cmd="$cmd -race"
fi

if [ "$cover" == true ]; then
  cmd="$cmd -coverprofile=coverage.txt -covermode=atomic -coverpkg=./...,./go-ethereum/..."
fi

if [ "$test_state_scheme" != "" ]; then
    cmd="$cmd -args -- --test_state_scheme=$test_state_scheme --test_loglevel=8"
else
    cmd="$cmd -args -- --test_loglevel=8" # Use error log level, which is the value 8 in the slog level enum for tests.
fi

if [ "$execution_consensus_jsonrpc_interconnect" == true ]; then
    cmd="$cmd --execution_consensus_jsonrpc_interconnect=true"
fi

if [ "$log" == true ]; then
    cmd="$cmd > >(stdbuf -oL tee -a full.log | grep -vE \"DEBUG|TRACE|INFO|seal\")"
else
    cmd="$cmd | grep -vE \"DEBUG|TRACE|INFO|seal\""
fi

echo ""
echo "$cmd"

if ! eval "$cmd"; then
  exit 1
fi
