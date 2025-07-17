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
      # Espresso Change: to expedite the CI, we always disable this flag
      cover=false
      shift
      ;;
    *)
      echo "Invalid argument: $1"
      exit 1
      ;;
  esac
done

###### Espresso
# 1. First ensure the library path is set
export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}:$(pwd)/target/lib"

# 2. Verify the library exists
if [[ ! -f "$(pwd)/target/lib/libespresso_crypto_helper-x86_64-unknown-linux-gnu.so" ]]; then
    echo "Error: libespresso_crypto_helper-x86_64-unknown-linux-gnu.so not found in $(pwd)/target/lib"
    exit 1
fi

skip_tests=$(grep -vE '^\s*#|^\s*$' ci_skip_tests | tr '\n' '|' | sed 's/|$//')
######


packages=$(go list ./...)
for package in $packages; do
  cmd="stdbuf -oL gotestsum --format short-verbose --packages=\"$package\" --rerun-fails=2 --no-color=false --"
  cmd="$cmd -skip \"$skip_tests\""
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

  if [ "$test_state_scheme" != "" ]; then
      cmd="$cmd -args -- --test_state_scheme=$test_state_scheme --test_loglevel=8"
  else
      cmd="$cmd -args -- --test_loglevel=8" # Use error log level, which is the value 8 in the slog level enum for tests.
  fi

  cmd="$cmd | grep -vE \"INFO|seal|TRACE|DEBUG\""

  echo ""
  echo running tests for "$package"
  echo "$cmd"

  if ! eval "$cmd"; then
    exit 1
  fi
done
