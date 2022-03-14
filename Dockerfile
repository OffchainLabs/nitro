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

FROM node:17-bullseye-slim as contracts-builder
RUN apt-get update && \
    apt-get install -y git python3 make g++
WORKDIR /workspace
COPY solgen/package.json solgen/yarn.lock solgen/
RUN cd solgen && yarn
COPY solgen solgen/
COPY Makefile .
RUN make build-solidity

FROM debian:bullseye-20211220 as wasm-base
WORKDIR /workspace
RUN apt-get update && apt-get install -y curl build-essential=12.9

FROM wasm-base as wasm-libs-builder
	# clang / lld used by soft-float wasm
RUN apt-get install -y clang=1:11.0-51+nmu5 lld=1:11.0-51+nmu5
    # pinned rust 1.58.1
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.59.0 --target x86_64-unknown-linux-gnu wasm32-unknown-unknown wasm32-wasi
COPY ./Makefile ./
COPY arbitrator/wasm-libraries arbitrator/wasm-libraries
COPY --from=brotli-wasm-export / target/
RUN . ~/.cargo/env && RUSTFLAGS='-C symbol-mangling-version=v0' make build-wasm-libs

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
COPY ./solgen/src/precompiles/ ./solgen/src/precompiles/
COPY ./solgen/gen.go ./solgen/package.json ./solgen/yarn.lock ./solgen/
COPY ./fastcache ./fastcache
COPY ./go-ethereum ./go-ethereum
COPY --from=brotli-wasm-export / target/
COPY --from=contracts-builder workspace/solgen/build/contracts/src/precompiles/ solgen/build/contracts/src/precompiles/
COPY --from=contracts-builder workspace/.make/ .make/
RUN PATH="$PATH:/usr/local/go/bin" make build-wasm-bin

FROM scratch as machine-exporter
COPY --from=wasm-libs-builder /workspace/target/machine/ machine/
COPY --from=wasm-bin-builder /workspace/target/machine/ machine/


FROM rust:1.57-slim-bullseye as prover-header-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make && \
    cargo install --force cbindgen
COPY arbitrator/Cargo.* arbitrator/cbindgen.toml arbitrator/
COPY ./Makefile ./
COPY arbitrator/prover arbitrator/prover
RUN make build-prover-header

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
RUN touch -a -m arbitrator/prover/src/lib.rs && \
    make build-prover-lib && make build-prover-bin

FROM scratch as prover-export
COPY --from=prover-builder /workspace/target/ /

FROM debian:bullseye-slim as module-root-calc
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y wabt
COPY --from=prover-export / target/
COPY --from=wasm-bin-builder /workspace/target/ target/
COPY --from=wasm-bin-builder /workspace/.make/ .make/
COPY --from=wasm-libs-builder /workspace/target/ target/
COPY --from=wasm-libs-builder /workspace/arbitrator/wasm-libraries/ arbitrator/wasm-libraries/
COPY --from=wasm-libs-builder /workspace/.make/ .make/
RUN target/bin/prover target/machine/replay.wasm --output-module-root -l target/machine/wasi_stub.wasm -l target/machine/soft-float.wasm -l target/machine/go_stub.wasm -l target/machine/host_io.wasm -l target/machine/brotli.wasm  > target/machine/module_root

FROM scratch as machine-export
COPY --from=module-root-calc /workspace/target/machine/ /machine


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
COPY --from=contracts-builder workspace/solgen/build/ solgen/build/
COPY --from=contracts-builder workspace/.make/ .make/
COPY --from=prover-header-export / target/
COPY --from=brotli-library-export / target/
COPY --from=prover-export / target/
RUN mkdir -p target/bin
# solgen was executed for just prcompiles previously.
RUN go run solgen/gen.go
RUN go build -o ./target/bin/node ./cmd/node
RUN go build -o ./target/bin/deploy ./cmd/deploy

FROM debian:bullseye-slim as nitro-node
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y wabt
COPY --from=node-builder /workspace/target/ target/
COPY --from=machine-export / target/
ENTRYPOINT [ "./target/bin/node" ]
