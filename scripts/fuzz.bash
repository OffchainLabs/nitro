#!/usr/bin/env bash

set -e

mydir=`dirname $0`
cd "$mydir"

function printusage {
    echo Usage: $0 --build \[--binary-path PATH\]
    echo "      "  $0 \<fuzzer-name\> \[--binary-path PATH\] \[--fuzzcache-path PATH\]  \[--nitro-path PATH\]
    echo
    echo fuzzer names:
    echo "   " FuzzPrecompiles
    echo "   " FuzzInboxMultiplexer
    echo "   " FuzzStateTransition
}

if [[ $# -eq 0 ]]; then
    printusage
    exit
fi

fuzz_executable=../target/bin/system_test.fuzz
binpath=../target/bin/
fuzzcachepath=../target/var/fuzz-cache
nitropath=../
run_build=false
test_group=""
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
        --build)
            run_build=true
            shift
            ;;
        FuzzPrecompiles | FuzzStateTransition)
            if [[ ! -z "$test_name" ]]; then
                echo can only run one fuzzer at a time
                exit 1
            fi
            test_group=system_tests
            test_name=$1
            shift
            ;;
        FuzzInboxMultiplexer)
            if [[ ! -z "$test_name" ]]; then
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

if $run_build; then
    for build_group in system_tests arbstate; do
        go test -c ${nitropath}/${build_group} -fuzz Fuzz -o "$binpath"/${build_group}.fuzz
    done
fi

if [[ ! -z $test_group ]]; then
    "$binpath"/${test_group}.fuzz -test.run "^$" -test.fuzzcachedir "$fuzzcachepath" -test.fuzz $test_name
fi
