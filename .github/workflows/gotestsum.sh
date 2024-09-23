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
race=false
cover=false
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
    --race)
      race=true
      shift
      ;;
    --cover)
      cover=true
      shift
      ;;
    *)
      echo "Invalid argument: $1"
      exit 1
      ;;
  esac
done

packages=$(go list ./...)
for package in $packages; do
  cmd="stdbuf -oL gotestsum --format short-verbose --packages=\"$package\" --rerun-fails=2 --no-color=false --"

  if [ "$timeout" != "" ]; then
    cmd="$cmd -timeout $timeout"
  fi

  if [ "$tags" != "" ]; then
    cmd="$cmd -tags=$tags"
  fi

  if [ "$run" != "" ]; then
    cmd="$cmd -run=$run"
  fi

  if [ "$race" == true ]; then
    cmd="$cmd -race"
  fi

  if [ "$cover" == true ]; then
    cmd="$cmd -coverprofile=coverage.txt -covermode=atomic -coverpkg=./...,./go-ethereum/..."
  fi

  cmd="$cmd > >(stdbuf -oL tee -a full.log | grep -vE \"INFO|seal\")"

  echo ""
  echo running tests for "$package"
  echo "$cmd"

  if ! eval "$cmd"; then
    exit 1
  fi
done
