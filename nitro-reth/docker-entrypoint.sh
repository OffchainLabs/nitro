#!/bin/bash
set -e

# Generate JWT secret if it doesn't exist
JWT_PATH="${JWT_SECRET_PATH:-/data/jwt.hex}"
if [ ! -f "$JWT_PATH" ]; then
    mkdir -p "$(dirname "$JWT_PATH")"
    openssl rand -hex 32 > "$JWT_PATH"
    echo "Generated JWT secret at $JWT_PATH"
fi

exec arb-reth "$@"
