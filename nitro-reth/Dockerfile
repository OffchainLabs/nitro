# Stage 1: Builder
FROM rust:1.93-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    clang \
    libclang-dev \
    cmake \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy workspace manifests first for dependency caching
COPY Cargo.toml Cargo.lock ./
COPY crates/ crates/
COPY bin/ bin/

# Build release binary
RUN cargo build --release -p arb-reth

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libssl3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /build/target/release/arb-reth /usr/local/bin/arb-reth

# Copy genesis files
COPY genesis/ /genesis/

EXPOSE 8545 8551

# Health check on auth RPC port
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 \
    CMD bash -c '</dev/tcp/localhost/8551' || exit 1

ENTRYPOINT ["arb-reth"]
CMD ["node"]
