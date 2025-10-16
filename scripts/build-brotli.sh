#!/usr/bin/env bash

set -euo pipefail

mydir=$(cd "$(dirname "$0")"; pwd)
cd "$mydir"

BUILD_WASM=false
BUILD_LOCAL=false
BUILD_SOFTFLOAT=false
USE_DOCKER=false
TARGET_DIR=../target/
SOURCE_DIR=../brotli
NITRO_DIR=../

usage(){
    echo "brotli builder for arbitrum"
    echo
    echo "usage: $0 [options]"
    echo
    echo "use one or more of:"
    echo " -w     build wasm (uses emscripten)"
    echo " -l     build local"
    echo " -f     build soft-float"
    echo
    echo "to avoid dependencies you might want:"
    echo " -d     build inside docker container"
    echo
    echo "Other options:"
    echo " -s     source dir default: $SOURCE_DIR"
    echo " -t     target dir default: $TARGET_DIR"
    echo " -n     nitro dir default: $NITRO_DIR"
    echo " -h     help"
    echo
    echo "all relative paths are relative to script location"
}

while getopts "n:s:t:c:D:wldhf" option; do
    case $option in
        h)
            usage
            exit
            ;;
        w)
            BUILD_WASM=true
            ;;
        l)
            BUILD_LOCAL=true
            ;;
        f)
            BUILD_SOFTFLOAT=true
            ;;
        d)
            USE_DOCKER=true
            ;;
        t)
            TARGET_DIR="$OPTARG"
            ;;
        n)
            NITRO_DIR="$OPTARG"
            ;;
        s)
            SOURCE_DIR="$OPTARG"
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

if ! $BUILD_WASM && ! $BUILD_LOCAL && ! $BUILD_SOFTFLOAT; then
    usage
    exit 1
fi

if [ ! -d "$TARGET_DIR" ]; then
    mkdir -p "${TARGET_DIR}/lib"
    ln -sf "lib" "${TARGET_DIR}/lib64" # Fedora build
fi
TARGET_DIR_ABS=$(cd -P "$TARGET_DIR"; pwd)

docker_build() {
    local target="$1"
    DOCKER_BUILDKIT=1 docker build --target "$target" -o type=local,dest="$TARGET_DIR_ABS" "${NITRO_DIR}"
}

if $USE_DOCKER; then
    $BUILD_WASM && docker_build "brotli-wasm-export"
    $BUILD_LOCAL && docker_build "brotli-library-export"
    $BUILD_SOFTFLOAT && docker_build "wasm-libs-export"
    exit 0
fi

cd "$SOURCE_DIR"

cmake_build() {
    local build_type="$1"
    shift
    local cmake_flags=("$@")

    local build_dir="buildfiles/build-$build_type"
    local install_dir="$TARGET_DIR_ABS"

    if [ "$build_type" = "wasm" ]; then
        mkdir -p buildfiles/install-wasm
        install_dir=$(cd -P buildfiles/install-wasm; pwd)
    fi

    mkdir -p "$build_dir"
    cd "$build_dir"

    cmake ../../ -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
                 -DCMAKE_BUILD_TYPE=Release \
                 -DCMAKE_INSTALL_PREFIX="$install_dir" \
                 "${cmake_flags[@]}"

    make -j
    make install

    if [ "$build_type" = "wasm" ]; then
        cp -rv "$install_dir/lib" "$TARGET_DIR_ABS/lib-wasm"
    fi

    cd ../..
}

$BUILD_WASM && cmake_build "wasm" -DCMAKE_C_COMPILER=emcc -DCMAKE_C_FLAGS=-fPIC -DCMAKE_AR="$(which emar)" -DCMAKE_RANLIB="$(which touch)"
$BUILD_LOCAL && cmake_build "local"
