#!/usr/bin/env bash

set -e

mydir=$(dirname "$0")
cd "$mydir"

function printusage {
    echo Usage: "$0" --build \[--binary-path PATH\]
    echo "      "  "$0" \<fuzzer-name\> \[--binary-path PATH\] \[--fuzzcache-path PATH\]  \[--nitro-path PATH\] \[--duration DURATION\]
    echo
    echo fuzzer names:
    echo "   " FuzzPrecompiles
    echo "   " FuzzInboxMultiplexer
    echo "   " FuzzStateTransition
    echo
    echo "   " duration in minutes
}

if [[ $# -eq 0 ]]; then
    printusage
    exit
fi

binpath=../target/bin/
fuzzcachepath=../target/var/fuzz-cache
nitropath=../
run_build=false
test_group=""
duration=60
while [[ $# -gt 0 ]]; do
    case $1 in
        --nitro-path)
            nitropath="$2"/
            if [[ ! -d "$nitropath" ]]; then
                echo must supply valid path for nitro-path
                exit 1
            fi
            shift
            shift
            ;;
        --binary-path)
            binpath="$2"/
            if [[ ! -d "$binpath" ]]; then
                echo must supply valid path for binary-path
                exit 1
            fi
            shift
            shift
            ;;
        --fuzzcache-path)
            fuzzcachepath="$2"
            if [[ ! -d "$binpath" ]]; then
                echo must supply valid path for fuzzcache-path
                exit 1
            fi
            shift
            shift
            ;;
        --duration)
            duration="$2"
            if ! [[ "$duration" =~ ^[0-9]+$ ]]; then
                echo "Invalid timeout duration. Please specify positive integer (in minutes)"
                exit 1
            fi
            shift
            shift
            ;;
        --build)
            run_build=true
            shift
            ;;
        FuzzPrecompiles | FuzzStateTransition)
            if [[ -n "$test_name" ]]; then
                echo can only run one fuzzer at a time
                exit 1
            fi
            test_group=system_tests
            test_name=$1
            shift
            ;;
        FuzzInboxMultiplexer)
            if [[ -n "$test_name" ]]; then
                echo can only run one fuzzer at a time
                exit 1
            fi
            test_group=arbstate
            test_name=$1
            shift
            ;;
        *)
            printusage
            exit
    esac
done

if [[ "$run_build" == "false" && -z "$test_group" ]]; then
    echo you must specify either --build flag or fuzzer-name
    printusage
fi

if $run_build; then
    for build_group in system_tests arbstate; do
        go test -c "${nitropath}"/${build_group} -fuzz Fuzz -o "$binpath"/${build_group}.fuzz
    done
fi

if [[ -n $test_group ]]; then
    timeout "$((60 * duration))" "$binpath"/${test_group}.fuzz -test.run "^$" -test.fuzzcachedir "$fuzzcachepath" -test.fuzz "$test_name" || exit_status=$?
fi

if  [ -n "$exit_status" ] && [ "$exit_status" -ne 0 ] && [ "$exit_status" -ne 124 ]; then
    echo "Fuzzing failed."
    exit "$exit_status"
fi

echo "Fuzzing succeeded."
