FROM node:17-bullseye-slim as contracts-builder
RUN apt-get update && \
    apt-get install -y git
WORKDIR /app
COPY solgen/package.json solgen/
RUN cd solgen && yarn
COPY solgen solgen/
RUN cd solgen && yarn build

FROM rust:1.57-slim-bullseye as arbitrator-builder
WORKDIR /workspace
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y make
RUN cargo install --force cbindgen
COPY arbitrator/Cargo.lock arbitrator/
COPY arbitrator/Cargo.toml arbitrator/
COPY arbitrator/prover/Cargo.toml arbitrator/prover/
RUN mkdir arbitrator/prover/src && \
    echo "fn test() {}" > arbitrator/prover/src/lib.rs && \
    cargo build --manifest-path arbitrator/Cargo.toml --release --lib
COPY ./Makefile ./
COPY arbitrator arbitrator/
RUN touch -a -m arbitrator/prover/src/lib.rs && \
    make build-node-rust-deps

FROM golang:1.17-bullseye as node-builder
COPY go.mod go.sum /workspace/
WORKDIR /workspace
COPY go.mod go.sum ./
COPY go-ethereum/go.mod go-ethereum/go.sum go-ethereum/
COPY fastcache/go.mod fastcache/go.sum fastcache/
RUN go mod download
COPY --from=contracts-builder app/solgen/artifacts solgen/artifacts/
COPY solgen/gen.go solgen/
COPY go-ethereum go-ethereum/
RUN mkdir -p solgen/go/ && \
	go run -v solgen/gen.go
COPY . ./
COPY --from=arbitrator-builder /workspace/arbitrator/target/env arbitrator/target/env/
RUN go build -v -o bin ./cmd/node ./cmd/deploy
#
FROM debian:bullseye-slim
COPY --from=node-builder /workspace/bin .
ENTRYPOINT [ "./node" ]
