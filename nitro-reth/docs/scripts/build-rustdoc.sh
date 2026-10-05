#!/usr/bin/env bash
# Build rustdoc and inject it into the Vocs dist directory.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCS_DIR="$SCRIPT_DIR/.."

echo "Building rustdoc..."
cd "$ROOT_DIR"
RUSTDOCFLAGS="--enable-index-page -Zunstable-options -A rustdoc::broken-intra-doc-links" \
  cargo +nightly doc --workspace --no-deps

echo "Injecting rustdoc into Vocs dist..."
DIST_DIR="$DOCS_DIR/docs/dist"
mkdir -p "$DIST_DIR/rustdoc"
cp -r "$ROOT_DIR/target/doc/." "$DIST_DIR/rustdoc/"

echo "Rustdoc injected at /rustdoc/"
