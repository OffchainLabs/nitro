# Pre-built binary Dockerfile
# Build the binary on the host first:
#   CARGO_TARGET_DIR=/data/target cargo build --release -p arb-reth
#   cp /data/target/release/arb-reth ./arb-reth
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libssl3 \
    ca-certificates \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# Copy pre-built binary
COPY arb-reth /usr/local/bin/arb-reth

# Copy genesis files and entrypoint
COPY genesis/ /genesis/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

EXPOSE 8545 8551

HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 \
    CMD bash -c '</dev/tcp/localhost/8551' || exit 1

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["node"]
