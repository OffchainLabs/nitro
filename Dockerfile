FROM emscripten/emsdk:3.1.7 as brotli-wasm-builder
WORKDIR /workspace
COPY build-brotli.sh .
COPY brotli brotli
RUN apt-get update && \
    apt-get install -y cmake make git && \
    # pinned emsdk 3.1.7 (in docker image)
    ./build-brotli.sh -w -t install/

FROM scratch as brotli-wasm-export
COPY --from=brotli-wasm-builder /workspace/install/ /

FROM debian:bullseye-slim as brotli-library-builder
WORKDIR /workspace
COPY build-brotli.sh .
COPY brotli brotli
RUN apt-get update && \
    apt-get install -y cmake make gcc git && \
    ./build-brotli.sh -l -t install/

FROM scratch as brotli-library-export
COPY --from=brotli-library-builder /workspace/install/ /

FROM node:16-bullseye-slim as contracts-builder
RUN apt-get update && \
    apt-get install -y git python3 make g++
WORKDIR /workspace
COPY contracts/package.json contracts/yarn.lock contracts/
RUN cd contracts && yarn
COPY contracts contracts/
COPY Makefile .
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-solidity

FROM debian:bullseye-20211220 as wasm-base
WORKDIR /workspace
RUN apt-get update && apt-get install -y curl build-essential=12.9

FROM wasm-base as wasm-libs-builder
	# clang / lld used by soft-float wasm
RUN apt-get install -y clang=1:11.0-51+nmu5 lld=1:11.0-51+nmu5
    # pinned rust 1.60.0
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.60.0 --target x86_64-unknown-linux-gnu wasm32-unknown-unknown wasm32-wasi
COPY ./Makefile ./
COPY arbitrator/wasm-libraries arbitrator/wasm-libraries
COPY --from=brotli-wasm-export / target/
RUN . ~/.cargo/env && NITRO_BUILD_IGNORE_TIMESTAMPS=1 RUSTFLAGS='-C symbol-mangling-version=v0' make build-wasm-libs

FROM wasm-base as wasm-bin-builder
    # pinned go version
RUN curl -L https://golang.org/dl/go1.17.8.linux-`dpkg --print-architecture`.tar.gz | tar -C /usr/local -xzf -
COPY ./Makefile ./go.mod ./go.sum ./
COPY ./arbcompress ./arbcompress
COPY ./arbos ./arbos
COPY ./arbstate ./arbstate
COPY ./blsSignatures ./blsSignatures
COPY ./cmd/replay ./cmd/replay
COPY ./precompiles ./precompiles
COPY ./statetransfer ./statetransfer
COPY ./util ./util
COPY ./wavmio ./wavmio
COPY ./zeroheavy ./zeroheavy
COPY ./contracts/src/precompiles/ ./contracts/src/precompiles/
COPY ./contracts/package.json ./contracts/yarn.lock ./contracts/
COPY ./solgen/gen.go ./solgen/
COPY ./fastcache ./fastcache
COPY ./go-ethereum ./go-ethereum
COPY --from=brotli-wasm-export / target/
COPY --from=contracts-builder workspace/contracts/build/contracts/src/precompiles/ contracts/build/contracts/src/precompiles/
COPY --from=contracts-builder workspace/.make/ .make/
RUN PATH="$PATH:/usr/local/go/bin" NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-wasm-bin

FROM rust:1.57-slim-bullseye as prover-header-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make && \
    cargo install --force cbindgen
COPY arbitrator/Cargo.* arbitrator/cbindgen.toml arbitrator/
COPY ./Makefile ./
COPY arbitrator/prover arbitrator/prover
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-header

FROM scratch as prover-header-export
COPY --from=prover-header-builder /workspace/target/ /

FROM rust:1.57-slim-bullseye as prover-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make
COPY arbitrator/Cargo.* arbitrator/
COPY arbitrator/prover/Cargo.toml arbitrator/prover/
RUN mkdir arbitrator/prover/src && \
    echo "fn test() {}" > arbitrator/prover/src/lib.rs && \
    cargo build --manifest-path arbitrator/Cargo.toml --release --lib
COPY ./Makefile ./
COPY arbitrator/prover arbitrator/prover
RUN touch -a -m arbitrator/prover/src/lib.rs
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-lib
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-bin

FROM scratch as prover-export
COPY --from=prover-builder /workspace/target/ /

FROM debian:bullseye-slim as module-root-calc
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y wabt make
COPY --from=prover-export / target/
COPY --from=wasm-bin-builder /workspace/target/ target/
COPY --from=wasm-bin-builder /workspace/.make/ .make/
COPY --from=wasm-libs-builder /workspace/target/ target/
COPY --from=wasm-libs-builder /workspace/arbitrator/wasm-libraries/ arbitrator/wasm-libraries/
COPY --from=wasm-libs-builder /workspace/.make/ .make/
COPY ./Makefile ./
COPY ./arbitrator ./arbitrator
COPY ./solgen ./solgen
COPY ./contracts ./contracts
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-replay-env

FROM debian:bullseye-slim as machine-versions
RUN apt-get update && apt-get install -y unzip wget
WORKDIR /workspace/machines
# Download old WASM module roots
#RUN bash -c 'r=0x21f708e444c3afb7689fa5d0737b3942fd19012c0081d359ba3d59b7643d7810; mkdir $r && ln -sfT $r latest && cd $r && echo $r > module-root.txt && wget https://github.com/OffchainLabs/nitro/releases/download/devnet-consensus-v1/machine.wavm.br'
RUN bash -c 'r=0xb7905959ec167e0777bbbd6c339b0c98d676729cb502722aa01a34964f817ca3; mkdir $r && ln -sfT $r latest && cd $r && echo $r > module-root.txt && wget https://github.com/OffchainLabs/nitro/releases/download/devnet-consensus-v2/machine.wavm.br'
RUN bash -c 'r=0xdd45c240cfe6624a7dfd0b0c965bf21301ebeea7a6a2a696781a6e03b04dc288; mkdir $r && ln -sfT $r latest && cd $r && echo $r > module-root.txt && wget https://github.com/OffchainLabs/nitro/releases/download/devnet-consensus-v3/machine.wavm.br'

FROM golang:1.17-bullseye as node-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y protobuf-compiler wabt
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY . ./
COPY --from=contracts-builder workspace/contracts/build/ contracts/build/
COPY --from=contracts-builder workspace/.make/ .make/
COPY --from=prover-header-export / target/
COPY --from=brotli-library-export / target/
COPY --from=prover-export / target/
RUN mkdir -p target/bin
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build

FROM debian:bullseye-slim as nitro-node-slim
WORKDIR /home/user
COPY --from=node-builder /workspace/target/bin/nitro /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/relay /usr/local/bin/
COPY --from=machine-versions /workspace/machines /home/user/target/machines
USER root
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    wabt && \
    useradd -ms /bin/bash user && \
    mkdir -p /home/user/l1keystore && \
    mkdir -p /home/user/.arbitrum/local/nitro && \
    chown -R user:user /home/user && \
    chmod -R 555 /home/user/target/machines && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/*

USER user
WORKDIR /home/user/
ENTRYPOINT [ "/usr/local/bin/nitro" ]

FROM nitro-node-slim as nitro-node
USER root
COPY --from=node-builder /workspace/target/bin/daserver /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/datool /usr/local/bin/
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    curl procps jq rsync \
    node-ws vim-tiny python3 \
    dnsutils && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/*

USER user

FROM nitro-node as nitro-node-dev
USER root
# Copy in latest WASM module root
RUN rm -f /home/user/target/machines/latest
COPY --from=node-builder /workspace/target/bin/deploy /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/seq-coordinator-invalidate /usr/local/bin/
COPY --from=module-root-calc /workspace/target/machines/latest/machine.wavm.br /home/user/target/machines/latest/
COPY --from=module-root-calc /workspace/target/machines/latest/until-host-io-state.bin /home/user/target/machines/latest/
COPY --from=module-root-calc /workspace/target/machines/latest/module-root.txt /home/user/target/machines/latest/
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    sudo && \
    chmod -R 555 /home/user/target/machines && \
    adduser user sudo && \
    echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/*

USER user

FROM nitro-node as nitro-node-default
# Just to ensure nitro-node-dist is default
