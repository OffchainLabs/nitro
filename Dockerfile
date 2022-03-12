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
WORKDIR /app
COPY solgen/package.json solgen/yarn.lock solgen/
RUN cd solgen && yarn
COPY solgen solgen/
RUN cd solgen && yarn build

FROM debian:bullseye-20211220 as wasm-base
WORKDIR /workspace
RUN apt-get update && apt-get install -y curl build-essential=12.9

FROM wasm-base as wasm-libs-builder
	# clang / lld used by soft-float wasm
RUN apt-get install -y clang=1:11.0-51+nmu5 lld=1:11.0-51+nmu5 
    # pinned rust 1.58.1
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.58.1 --target x86_64-unknown-linux-gnu wasm32-unknown-unknown wasm32-wasi
RUN . ~/.cargo/env && cargo install cbindgen --version =0.20.0
COPY ./Makefile ./
COPY ./brotli/Makefile ./brotli/Makefile
COPY arbitrator/wasm-libraries arbitrator/wasm-libraries
COPY --from=brotli-wasm-export / target/
RUN . ~/.cargo/env && make build-wasm-libs

FROM wasm-base as wasm-bin-builder
    # pinned go version
RUN curl -L https://golang.org/dl/go1.17.8.linux-amd64.tar.gz | tar -C /usr/local -xzf -
COPY ./Makefile ./
COPY ./go.* ./
COPY ./arbcompress ./arbcompress
COPY ./arbos ./arbos
COPY ./arbstate ./arbstate
COPY ./blsSignatures ./blsSignatures
COPY ./cmd/replay ./cmd/replay
COPY ./das ./das
COPY ./precompiles ./precompiles
COPY ./statetransfer ./statetransfer
COPY ./util ./util
COPY ./wavmio ./wavmio
COPY ./solgen ./solgen
COPY ./fastcache ./fastcache
COPY ./go-ethereum ./go-ethereum
COPY --from=brotli-wasm-export / target/
COPY --from=contracts-builder app/solgen/build solgen/build/
RUN mkdir -p solgen/go
RUN PATH="$PATH:/usr/local/go/bin" go run solgen/gen.go
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

FROM rust:1.57-slim-bullseye as prover-lib-builder
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
    make build-prover-lib

FROM scratch as prover-lib-export
COPY --from=prover-lib-builder /workspace/target/ /

FROM golang:1.17-bullseye as node-builder
COPY go.mod go.sum /workspace/
WORKDIR /workspace
COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY --from=contracts-builder app/solgen/build/ solgen/build/
COPY solgen/gen.go solgen/
COPY go-ethereum go-ethereum/
RUN mkdir -p solgen/go/ && \
	go run -v solgen/gen.go
COPY . ./
COPY --from=prover-header-export / target/
COPY --from=prover-lib-export / target/
COPY --from=brotli-library-export / target/
RUN mkdir -p target/bin && \
    go build -v -o target/bin ./cmd/node ./cmd/deploy && \
    GOOS=js GOARCH=wasm go build -o res/target/lib/replay.wasm ./cmd/replay/...

FROM debian:bullseye-slim as nitro-node
COPY --from=node-builder /workspace/target/ target/
COPY --from=machine-exporter / target/
ENTRYPOINT [ "./target/bin/node" ]
