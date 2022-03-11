FROM node:17-bullseye-slim as contracts-builder
RUN apt-get update && \
    apt-get install -y git python3 make g++
WORKDIR /app
COPY solgen/package.json solgen/yarn.lock solgen/
RUN cd solgen && yarn
COPY solgen solgen/
RUN cd solgen && yarn build

FROM rust:1.57-slim-bullseye as wasm-lib-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make clang lld && \
    rustup target add wasm32-unknown-unknown && \
    rustup target add wasm32-wasi
COPY ./Makefile ./
COPY arbitrator/wasm-libraries arbitrator/wasm-libraries/
RUN make build-wasm-libs

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
COPY --from=prover-header-builder /workspace/target/ target/
COPY --from=prover-lib-builder /workspace/target/ target/
RUN mkdir -p target/bin && \
    go build -v -o target/bin ./cmd/node ./cmd/deploy ./cmd/relay && \
    GOOS=js GOARCH=wasm go build -o res/target/lib/replay.wasm ./cmd/replay/...

FROM debian:bullseye-slim as nitro-node
COPY --from=node-builder /workspace/target/ target/
COPY --from=wasm-lib-builder /workspace/target/ target/
ENTRYPOINT [ "./target/bin/node" ]
