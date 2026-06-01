# syntax=docker/dockerfile:1
# BINARY_STAGE selects the binary source: "compile" (default) builds it here,
# "prebuilt" takes it from the build context. The release pipeline uses
# "prebuilt" so per-arch images reuse the binaries already built rather than
# recompiling (which is slow and fails on emulated/low-memory arm64 runners).
ARG BINARY_STAGE=compile

# Stage 1: Compile
FROM rust:1.93-bookworm AS compile

RUN apt-get update && apt-get install -y \
    clang \
    libclang-dev \
    cmake \
    git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY Cargo.toml Cargo.lock ./
COPY .cargo/ .cargo/
COPY crates/ crates/
COPY bin/ bin/
COPY .gitmodules ./
COPY brotli/ brotli/

RUN cargo build --release --locked -p arb-reth --bin arb-reth \
    && cp target/release/arb-reth /arb-reth

# Prebuilt binary supplied via the build context (release pipeline).
FROM scratch AS prebuilt
COPY arb-reth /arb-reth

# Resolve the binary source selected by BINARY_STAGE.
FROM ${BINARY_STAGE} AS binsrc

# Stage 2: Runtime
# Ubuntu 24.04 to match the glibc the release binaries are built against
# (the binary jobs run on ubuntu-24.04 runners).
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    libssl3t64 \
    ca-certificates \
    openssl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=binsrc --chmod=755 /arb-reth /usr/local/bin/arb-reth

# Copy genesis files and entrypoint
COPY genesis/ /genesis/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

EXPOSE 8545 8551

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 \
    CMD bash -c '</dev/tcp/localhost/8551' || exit 1

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["node"]
