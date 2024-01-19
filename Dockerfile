FROM debian:bullseye-slim as brotli-wasm-builder
WORKDIR /workspace
RUN apt-get update && \
    apt-get install -y cmake make git lbzip2 python3 xz-utils && \
    git clone https://github.com/emscripten-core/emsdk.git && \
    cd emsdk && \
    ./emsdk install 3.1.7 && \
    ./emsdk activate 3.1.7
COPY scripts/build-brotli.sh scripts/
COPY brotli brotli
RUN cd emsdk && . ./emsdk_env.sh && cd .. && ./scripts/build-brotli.sh -w -t /workspace/install/

FROM scratch as brotli-wasm-export
COPY --from=brotli-wasm-builder /workspace/install/ /

FROM debian:bullseye-slim as brotli-library-builder
WORKDIR /workspace
COPY scripts/build-brotli.sh scripts/
COPY brotli brotli
RUN apt-get update && \
    apt-get install -y cmake make gcc git && \
    ./scripts/build-brotli.sh -l -t /workspace/install/

FROM scratch as brotli-library-export
COPY --from=brotli-library-builder /workspace/install/ /

FROM node:16-bullseye-slim as contracts-builder
RUN apt-get update && \
    apt-get install -y git python3 make g++
WORKDIR /workspace
COPY contracts/package.json contracts/yarn.lock contracts/
RUN cd contracts && yarn install
COPY contracts contracts/
COPY Makefile .
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-solidity

FROM debian:bullseye-20211220 as wasm-base
WORKDIR /workspace
RUN apt-get update && apt-get install -y curl build-essential=12.9

FROM wasm-base as wasm-libs-builder
	# clang / lld used by soft-float wasm
RUN apt-get install -y clang=1:11.0-51+nmu5 lld=1:11.0-51+nmu5
    # pinned rust 1.65.0
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.68.2 --target x86_64-unknown-linux-gnu wasm32-unknown-unknown wasm32-wasi
COPY ./Makefile ./
COPY arbitrator/arbutil arbitrator/arbutil
COPY arbitrator/wasm-libraries arbitrator/wasm-libraries
COPY --from=brotli-wasm-export / target/
RUN . ~/.cargo/env && NITRO_BUILD_IGNORE_TIMESTAMPS=1 RUSTFLAGS='-C symbol-mangling-version=v0' make build-wasm-libs

FROM scratch as wasm-libs-export
COPY --from=wasm-libs-builder /workspace/ /

FROM wasm-base as wasm-bin-builder
    # pinned go version
RUN curl -L https://golang.org/dl/go1.20.linux-`dpkg --print-architecture`.tar.gz | tar -C /usr/local -xzf -
COPY ./Makefile ./go.mod ./go.sum ./
COPY ./arbcompress ./arbcompress
COPY ./arbos ./arbos
COPY ./arbstate ./arbstate
COPY ./arbutil ./arbutil
COPY ./gethhook ./gethhook
COPY ./blsSignatures ./blsSignatures
COPY ./cmd/chaininfo ./cmd/chaininfo
COPY ./cmd/replay ./cmd/replay
COPY ./das/dastree ./das/dastree
COPY ./das/eigenda ./das/eigenda
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
COPY --from=contracts-builder workspace/contracts/node_modules/@offchainlabs/upgrade-executor/build/contracts/src/UpgradeExecutor.sol/UpgradeExecutor.json contracts/
COPY --from=contracts-builder workspace/.make/ .make/
RUN PATH="$PATH:/usr/local/go/bin" NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-wasm-bin

FROM rust:1.68-slim-bullseye as prover-header-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make && \
    cargo install --force cbindgen
COPY arbitrator/Cargo.* arbitrator/cbindgen.toml arbitrator/
COPY ./Makefile ./
COPY arbitrator/arbutil arbitrator/arbutil
COPY arbitrator/prover arbitrator/prover
COPY arbitrator/jit arbitrator/jit
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-header

FROM scratch as prover-header-export
COPY --from=prover-header-builder /workspace/target/ /

FROM rust:1.68-slim-bullseye as prover-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make wget gpg software-properties-common zlib1g-dev libstdc++-10-dev wabt
RUN wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add - && \
    add-apt-repository 'deb http://apt.llvm.org/bullseye/ llvm-toolchain-bullseye-12 main' && \
    apt-get update && \
    apt-get install -y llvm-12-dev libclang-common-12-dev
COPY arbitrator/Cargo.* arbitrator/
COPY arbitrator/arbutil arbitrator/arbutil
COPY arbitrator/prover/Cargo.toml arbitrator/prover/
COPY arbitrator/jit/Cargo.toml arbitrator/jit/
RUN mkdir arbitrator/prover/src arbitrator/jit/src && \
    echo "fn test() {}" > arbitrator/jit/src/lib.rs && \
    echo "fn test() {}" > arbitrator/prover/src/lib.rs && \
    cargo build --manifest-path arbitrator/Cargo.toml --release --lib && \
    rm arbitrator/jit/src/lib.rs
COPY ./Makefile ./
COPY arbitrator/prover arbitrator/prover
COPY arbitrator/jit arbitrator/jit
COPY --from=brotli-library-export / target/
RUN touch -a -m arbitrator/prover/src/lib.rs
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-lib
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build-prover-bin
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make CARGOFLAGS="--features=llvm" build-jit

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
RUN apt-get update && apt-get install -y unzip wget curl
WORKDIR /workspace/machines
# Download WAVM machines
COPY ./scripts/download-machine.sh .
#RUN ./download-machine.sh consensus-v1-rc1 0xbb9d58e9527566138b682f3a207c0976d5359837f6e330f4017434cca983ff41
#RUN ./download-machine.sh consensus-v2.1 0x9d68e40c47e3b87a8a7e6368cc52915720a6484bb2f47ceabad7e573e3a11232
#RUN ./download-machine.sh consensus-v3 0x53c288a0ca7100c0f2db8ab19508763a51c7fd1be125d376d940a65378acaee7
#RUN ./download-machine.sh consensus-v3.1 0x588762be2f364be15d323df2aa60ffff60f2b14103b34823b6f7319acd1ae7a3
#RUN ./download-machine.sh consensus-v3.2 0xcfba6a883c50a1b4475ab909600fa88fc9cceed9e3ff6f43dccd2d27f6bd57cf
#RUN ./download-machine.sh consensus-v4 0xa24ccdb052d92c5847e8ea3ce722442358db4b00985a9ee737c4e601b6ed9876
#RUN ./download-machine.sh consensus-v5 0x1e09e6d9e35b93f33ed22b2bc8dc10bbcf63fdde5e8a1fb8cc1bcd1a52f14bd0
#RUN ./download-machine.sh consensus-v6 0x3848eff5e0356faf1fc9cafecb789584c5e7f4f8f817694d842ada96613d8bab
#RUN ./download-machine.sh consensus-v7 0x53dd4b9a3d807a8cbb4d58fbfc6a0857c3846d46956848cae0a1cc7eca2bb5a8
#RUN ./download-machine.sh consensus-v7.1 0x2b20e1490d1b06299b222f3239b0ae07e750d8f3b4dedd19f500a815c1548bbc
#RUN ./download-machine.sh consensus-v9 0xd1842bfbe047322b3f3b3635b5fe62eb611557784d17ac1d2b1ce9c170af6544
RUN ./download-machine.sh consensus-v10 0x6b94a7fc388fd8ef3def759297828dc311761e88d8179c7ee8d3887dc554f3c3
RUN ./download-machine.sh consensus-v10.1 0xda4e3ad5e7feacb817c21c8d0220da7650fe9051ece68a3f0b1c5d38bbb27b21
RUN ./download-machine.sh consensus-v10.2 0x0754e09320c381566cc0449904c377a52bd34a6b9404432e80afd573b67f7b17
RUN ./download-machine.sh consensus-v10.3 0xf559b6d4fa869472dabce70fe1c15221bdda837533dfd891916836975b434dec
RUN ./download-machine.sh consensus-v11 0xf4389b835497a910d7ba3ebfb77aa93da985634f3c052de1290360635be40c4a

FROM golang:1.20-bullseye as node-builder
WORKDIR /workspace
ARG version=""
ARG datetime=""
ARG modified=""
ENV NITRO_VERSION=$version
ENV NITRO_DATETIME=$datetime
ENV NITRO_MODIFIED=$modified
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y wabt
COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY . ./
COPY --from=contracts-builder workspace/contracts/build/ contracts/build/
COPY --from=contracts-builder workspace/contracts/node_modules/@offchainlabs/upgrade-executor/build/contracts/src/UpgradeExecutor.sol/UpgradeExecutor.json contracts/node_modules/@offchainlabs/upgrade-executor/build/contracts/src/UpgradeExecutor.sol/
COPY --from=contracts-builder workspace/.make/ .make/
COPY --from=prover-header-export / target/
COPY --from=brotli-library-export / target/
COPY --from=prover-export / target/
RUN mkdir -p target/bin
COPY .nitro-tag.txt /nitro-tag.txt
RUN NITRO_BUILD_IGNORE_TIMESTAMPS=1 make build

FROM node-builder as fuzz-builder
RUN mkdir fuzzers/
RUN ./scripts/fuzz.bash --build --binary-path /workspace/fuzzers/

FROM debian:bullseye-slim as nitro-fuzzer
COPY --from=fuzz-builder /workspace/fuzzers/*.fuzz /usr/local/bin/
COPY ./scripts/fuzz.bash /usr/local/bin
RUN mkdir /fuzzcache
ENTRYPOINT [ "/usr/local/bin/fuzz.bash", "--binary-path", "/usr/local/bin/", "--fuzzcache-path", "/fuzzcache" ]

FROM debian:bullseye-slim as nitro-node-slim
WORKDIR /home/user
COPY --from=node-builder /workspace/target/bin/nitro /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/relay /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/nitro-val /usr/local/bin/
COPY --from=node-builder /workspace/target/bin/seq-coordinator-manager /usr/local/bin/
COPY --from=machine-versions /workspace/machines /home/user/target/machines
USER root
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    ca-certificates \
    wabt && \
    /usr/sbin/update-ca-certificates && \
    useradd -s /bin/bash user && \
    mkdir -p /home/user/l1keystore && \
    mkdir -p /home/user/.arbitrum/local/nitro && \
    chown -R user:user /home/user && \
    chmod -R 555 /home/user/target/machines && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/* /var/cache/ldconfig/aux-cache /usr/lib/python3.9/__pycache__/ /usr/lib/python3.9/*/__pycache__/ /var/log/* && \
    nitro --version

USER user
WORKDIR /home/user/
ENTRYPOINT [ "/usr/local/bin/nitro" ]

FROM nitro-node-slim as nitro-node
USER root
COPY --from=prover-export /bin/jit                        /usr/local/bin/
COPY --from=node-builder  /workspace/target/bin/daserver  /usr/local/bin/
COPY --from=node-builder  /workspace/target/bin/datool    /usr/local/bin/
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    curl procps jq rsync \
    node-ws vim-tiny python3 \
    dnsutils && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/* /var/cache/ldconfig/aux-cache /usr/lib/python3.9/__pycache__/ /usr/lib/python3.9/*/__pycache__/ /var/log/* && \
    nitro --version

USER user

FROM nitro-node as nitro-node-dev
USER root
# Copy in latest WASM module root
RUN rm -f /home/user/target/machines/latest
COPY --from=prover-export /bin/jit                                         /usr/local/bin/
COPY --from=node-builder  /workspace/target/bin/deploy                     /usr/local/bin/
COPY --from=node-builder  /workspace/target/bin/seq-coordinator-invalidate /usr/local/bin/
COPY --from=module-root-calc /workspace/target/machines/latest/machine.wavm.br /home/user/target/machines/latest/
COPY --from=module-root-calc /workspace/target/machines/latest/until-host-io-state.bin /home/user/target/machines/latest/
COPY --from=module-root-calc /workspace/target/machines/latest/module-root.txt /home/user/target/machines/latest/
COPY --from=module-root-calc /workspace/target/machines/latest/replay.wasm /home/user/target/machines/latest/
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y \
    sudo && \
    chmod -R 555 /home/user/target/machines && \
    adduser user sudo && \
    echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /usr/share/doc/* /var/cache/ldconfig/aux-cache /usr/lib/python3.9/__pycache__/ /usr/lib/python3.9/*/__pycache__/ /var/log/* && \
    nitro --version

USER user

FROM nitro-node as nitro-node-default
# Just to ensure nitro-node-dist is default
