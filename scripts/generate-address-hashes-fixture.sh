#!/usr/bin/env bash
#
# generate-address-hashes-fixture.sh
#
# Produce a filtered-addresses-hashed-list.json fixture file
# of arbitrary size for dev/test-net / load testing.
#
# Real hashes (20 hardcoded addresses + any from --addresses) are computed
# with the production algorithm, selected by --hashing-scheme:
#   sha256-rawbytesinput (default): sha256(salt_16_raw_bytes || addr_20_raw_bytes)
#   sha256-stringinput            : sha256(salt_uuid_string + "::0x" + lowercase_addr_hex)
# (mirrors execution/gethexec/addressfilter/hash_store.go: HashStringInputWithPrefix /
# GetHashStringInputPrefix / HashRawBytesInput). The remainder is filler: zero-padded sequential
# counters formatted as 64-hex strings ("0x000...001", "0x000...002", ...),
# emitted in parallel by 8 awk workers writing to per-chunk temp files,
# then concatenated.
#
# Cross-platform: macOS and Linux.

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
SIZE="4GB"
OUT="filtered-addresses-hashed-list.json"
EXTRA_ADDRESSES=""
SEED=""
SALT_OVERRIDE=""
JOBS=8
HASHING_SCHEME="sha256-rawbytesinput"

# 20 hardcoded test addresses: 0x00...01 through 0x00...14.
readonly HARDCODED_ADDRESSES=(
    "0x0000000000000000000000000000000000000001"
    "0x0000000000000000000000000000000000000002"
    "0x0000000000000000000000000000000000000003"
    "0x0000000000000000000000000000000000000004"
    "0x0000000000000000000000000000000000000005"
    "0x0000000000000000000000000000000000000006"
    "0x0000000000000000000000000000000000000007"
    "0x0000000000000000000000000000000000000008"
    "0x0000000000000000000000000000000000000009"
    "0x000000000000000000000000000000000000000a"
    "0x000000000000000000000000000000000000000b"
    "0x000000000000000000000000000000000000000c"
    "0x000000000000000000000000000000000000000d"
    "0x000000000000000000000000000000000000000e"
    "0x000000000000000000000000000000000000000f"
    "0x0000000000000000000000000000000000000010"
    "0x0000000000000000000000000000000000000011"
    "0x0000000000000000000000000000000000000012"
    "0x0000000000000000000000000000000000000013"
    "0x0000000000000000000000000000000000000014"
)

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Generate filtered-addresses-hashed-list.json fixture file.

Options:
  -s, --size SIZE         Target file size, e.g. 4GB, 500MB. Suffix MB or GB
                          (binary units; 1 GB = 2^30 bytes). Default: 4GB.
  -o, --out PATH          Output file path. If it ends with "/" or is an
                          existing directory, the default filename
                          (filtered-addresses-hashed-list.json) is appended.
                          Default: ./filtered-addresses-hashed-list.json.
      --addresses LIST    Comma-separated extra addresses (0x...) to include
                          as real hashes alongside the 20 hardcoded ones.
      --salt UUID         Use this UUID as the hashing salt (8-4-4-4-12 hex).
                          Default: random (or derived from --seed if given).
      --seed STRING       Derive UUIDs deterministically from this seed (for
                          reproducible fixtures). Filler is always
                          counter-based and reproducible regardless.
      --hashing-scheme S  Hashing scheme for real hashes: sha256-rawbytesinput
                          (default) or sha256-stringinput. Written to the
                          file's hashing_scheme field.
      --jobs N            Number of parallel filler workers. Default: 8.
  -h, --help              Show this help.
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

# ---------------------------------------------------------------------------
# Cross-platform tool detection (sha256, stat flavor)
# ---------------------------------------------------------------------------
SHA256_CMD=""
STAT_FLAVOR=""

detect_tools() {
    local cmd
    for cmd in uuidgen tr awk printf; do
        command -v "$cmd" >/dev/null 2>&1 || die "required command not found: $cmd"
    done

    if command -v sha256sum >/dev/null 2>&1; then
        SHA256_CMD="sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        SHA256_CMD="shasum -a 256"
    elif command -v openssl >/dev/null 2>&1; then
        SHA256_CMD="openssl_sha256"
    else
        die "need one of: sha256sum, shasum, openssl"
    fi

    if stat -c%s /dev/null >/dev/null 2>&1; then
        STAT_FLAVOR="gnu"
    elif stat -f%z /dev/null >/dev/null 2>&1; then
        STAT_FLAVOR="bsd"
    else
        die "neither GNU (-c%s) nor BSD (-f%z) stat available"
    fi
}

sha256_hex() {
    if [[ "$SHA256_CMD" == "openssl_sha256" ]]; then
        openssl dgst -sha256 -r | awk '{print $1}'
    else
        $SHA256_CMD | awk '{print $1}'
    fi
}

# Raw-bytes scheme: sha256(16 salt bytes || 20 addr bytes), mirrors HashRawBytesInput.
hash_raw_bytes() {
    local salt_uuid="$1" addr_hex="$2"
    local salt_hex hex esc i
    salt_hex=$(printf '%s' "$salt_uuid" | tr -d '-')
    hex="${salt_hex}${addr_hex}"
    esc=""
    for (( i=0; i<${#hex}; i+=2 )); do
        esc+="\\x${hex:i:2}"
    done
    printf '%b' "$esc" | sha256_hex
}

file_size() {
    case "$STAT_FLAVOR" in
        gnu) stat -c%s "$1" ;;
        bsd) stat -f%z "$1" ;;
    esac
}

# uuidgen prints uppercase on macOS, lowercase on Linux. Production hashing
# uses uuid.UUID.String() which is RFC 4122 lowercase, so normalize.
lower_uuid() {
    uuidgen | tr '[:upper:]' '[:lower:]'
}

# When --seed is set, derive UUIDs deterministically so the whole file
# (UUIDs + real hashes + filler) is byte-identical across runs. The
# google/uuid Go parser accepts any 32-hex 8-4-4-4-12 string regardless
# of version/variant bits, so we don't need to set them.
seeded_uuid() {
    local label="$1"
    local h
    h=$(printf '%s::%s' "$SEED" "$label" | sha256_hex)
    printf '%s-%s-%s-%s-%s' \
        "${h:0:8}" "${h:8:4}" "${h:12:4}" "${h:16:4}" "${h:20:12}"
}

# ---------------------------------------------------------------------------
# Arg parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        -s|--size)        [[ $# -ge 2 ]] || die "$1 requires a value"; SIZE="$2"; shift 2 ;;
        -o|--out)         [[ $# -ge 2 ]] || die "$1 requires a value"; OUT="$2"; shift 2 ;;
        --addresses)      [[ $# -ge 2 ]] || die "$1 requires a value"; EXTRA_ADDRESSES="$2"; shift 2 ;;
        --salt)           [[ $# -ge 2 ]] || die "$1 requires a value"; SALT_OVERRIDE="$2"; shift 2 ;;
        --seed)           [[ $# -ge 2 ]] || die "$1 requires a value"; SEED="$2"; shift 2 ;;
        --hashing-scheme) [[ $# -ge 2 ]] || die "$1 requires a value"; HASHING_SCHEME="$2"; shift 2 ;;
        --jobs)           [[ $# -ge 2 ]] || die "$1 requires a value"
                          [[ "$2" =~ ^[1-9][0-9]*$ ]] || die "invalid --jobs: $2 (expected a positive integer)"
                          JOBS="$2"; shift 2 ;;
        -h|--help)        usage; exit 0 ;;
        *)                die "unknown argument: $1 (use --help)" ;;
    esac
done

case "$HASHING_SCHEME" in
    sha256-stringinput|sha256-rawbytesinput) ;;
    *) die "invalid --hashing-scheme: $HASHING_SCHEME (expected sha256-stringinput or sha256-rawbytesinput)" ;;
esac

detect_tools

# ---------------------------------------------------------------------------
# Parse size into bytes
# ---------------------------------------------------------------------------
parse_size() {
    local s="$1"
    local upper
    upper=$(printf '%s' "$s" | tr '[:lower:]' '[:upper:]')
    if [[ "$upper" =~ ^([0-9]+)(MB|GB)$ ]]; then
        local n="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"
        case "$unit" in
            MB) printf '%s' "$((n * 1048576))" ;;
            GB) printf '%s' "$((n * 1073741824))" ;;
        esac
    else
        die "invalid --size: $s (expected e.g. 500MB, 4GB)"
    fi
}

TARGET_BYTES=$(parse_size "$SIZE")

# ---------------------------------------------------------------------------
# Build the real-address list (hardcoded plus --addresses), validate, dedupe
# ---------------------------------------------------------------------------
combined=()
for a in "${HARDCODED_ADDRESSES[@]}"; do
    combined+=("$a")
done

if [[ -n "$EXTRA_ADDRESSES" ]]; then
    IFS=',' read -ra extras <<< "$EXTRA_ADDRESSES"
    for a in "${extras[@]}"; do
        a=$(printf '%s' "$a" | tr -d '[:space:]')
        [[ "$a" =~ ^0[xX][0-9a-fA-F]{40}$ ]] || die "invalid address: $a"
        a_lower=$(printf '%s' "${a:2}" | tr '[:upper:]' '[:lower:]')
        combined+=("0x${a_lower}")
    done
fi

# Dedupe while preserving first-seen order (awk idiom).
REAL_ADDRESSES=()
while IFS= read -r line; do
    REAL_ADDRESSES+=("$line")
done < <(printf '%s\n' "${combined[@]}" | awk 'NF && !seen[$0]++')

REAL_COUNT=${#REAL_ADDRESSES[@]}

# ---------------------------------------------------------------------------
# UUIDs and salt-derived hash prefix
# ---------------------------------------------------------------------------
if [[ -n "$SEED" ]]; then
    ID=$(seeded_uuid "id")
    EXTRACT_UUID=$(seeded_uuid "extract_uuid")
    SALT=$(seeded_uuid "salt")
else
    ID=$(lower_uuid)
    EXTRACT_UUID=$(lower_uuid)
    SALT=$(lower_uuid)
fi

if [[ -n "$SALT_OVERRIDE" ]]; then
    SALT=$(printf '%s' "$SALT_OVERRIDE" | tr '[:upper:]' '[:lower:]')
    [[ "$SALT" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] \
        || die "invalid --salt: expected UUID 8-4-4-4-12 hex, got: $SALT_OVERRIDE"
fi

# issued_at is RFC 3339 UTC with no fractional seconds (exactly 20 chars).
# Under --seed we use a fixed placeholder so seeded runs stay byte-identical.
if [[ -n "$SEED" ]]; then
    ISSUED_AT="1970-01-01T00:00:00Z"
else
    ISSUED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
fi

HASH_PREFIX="${SALT}::0x"

# ---------------------------------------------------------------------------
# Compute real hashes (mirrors hashAddress in hash_store.go: string-input or
# raw-bytes path, selected by --hashing-scheme)
# ---------------------------------------------------------------------------
REAL_HASHES=()
for addr in "${REAL_ADDRESSES[@]}"; do
    addr_hex="${addr:2}"
    if [[ "$HASHING_SCHEME" == "sha256-rawbytesinput" ]]; then
        h=$(hash_raw_bytes "$SALT" "$addr_hex")
    else
        h=$(printf '%s%s' "$HASH_PREFIX" "$addr_hex" | sha256_hex)
    fi
    REAL_HASHES+=("$h")
done

# ---------------------------------------------------------------------------
# Filler count from byte arithmetic.
#
# Layout (no whitespace between entries):
#   header        210 + len(scheme) bytes  ({"id":"<36>","extract_uuid":"<36>","salt":"<36>","issued_at":"<20>","hashing_scheme":"<scheme>","hashes":[)
#   each non-final hash   69 bytes  ("0x<64hex>",)
#   final hash            68 bytes  ("0x<64hex>")
#   footer                 2 bytes  (]})
#
#   total = HEADER_BYTES + 69*(N-1) + 68 + 2 = (HEADER_BYTES + 1) + 69*N
# (HEADER_BYTES is 228 for sha256-stringinput, 230 for sha256-rawbytesinput.)
# ---------------------------------------------------------------------------
HEADER_BYTES=$(( 210 + ${#HASHING_SCHEME} ))
TOTAL_HASHES=$(( (TARGET_BYTES - (HEADER_BYTES + 1)) / 69 ))
FILLER_COUNT=$(( TOTAL_HASHES - REAL_COUNT ))

if (( FILLER_COUNT < 1 )); then
    die "target size $SIZE too small for $REAL_COUNT real hashes plus 1 filler"
fi

# Number of bulk filler entries (the very last entry is written separately
# without a trailing comma so the JSON closes cleanly).
BULK_COUNT=$(( FILLER_COUNT - 1 ))

if (( JOBS < 1 )); then die "--jobs must be >= 1"; fi
if (( JOBS > BULK_COUNT && BULK_COUNT > 0 )); then JOBS=$BULK_COUNT; fi

# ---------------------------------------------------------------------------
# Parallel filler generation
#
# Each worker writes its slice of the counter range to a temp file as
#   "0x<64-zero-padded-counter>","0x<...>",..."0x<...>",
# (every entry followed by a comma — final no-comma entry is written by the
# main process). Counters are simple sequence values, never colliding with
# real sha256 hashes (collision probability ~2^-240). The temp directory
# lives under $TMPDIR (tmpfs on Linux), so chunk files stay in RAM.
# ---------------------------------------------------------------------------
TMPDIR_RUN=$(mktemp -d -t addrhashes.XXXXXX)
trap 'rm -rf "$TMPDIR_RUN"' EXIT INT TERM

PER_CHUNK=$(( BULK_COUNT / JOBS ))
REM=$(( BULK_COUNT % JOBS ))

PIDS=()
next_start=1
for (( k=0; k<JOBS; k++ )); do
    if (( k < REM )); then
        count=$(( PER_CHUNK + 1 ))
    else
        count=$PER_CHUNK
    fi
    chunk_file=$(printf '%s/chunk.%03d' "$TMPDIR_RUN" "$k")
    awk -v start="$next_start" -v n="$count" '
        BEGIN { for (i = 0; i < n; i++) printf "\"0x%064x\",", start + i }
    ' > "$chunk_file" &
    PIDS+=("$!")
    next_start=$(( next_start + count ))
done

# Counter for the final no-comma entry.
LAST_COUNTER=$next_start

fail=0
for pid in "${PIDS[@]}"; do
    wait "$pid" || fail=1
done
(( fail == 0 )) || die "filler worker failed"

# ---------------------------------------------------------------------------
# Stream-write the file: header → real hashes → chunk files in order →
# final no-comma hash → footer. All in a single redirect group so the
# output file is opened once.
# ---------------------------------------------------------------------------
# If --out is a directory (or ends with /), append the default filename.
if [[ "$OUT" == */ ]] || [[ -d "$OUT" ]]; then
    OUT="${OUT%/}/filtered-addresses-hashed-list.json"
fi
OUT_PATH="$OUT"
mkdir -p "$(dirname "$OUT_PATH")"

{
    printf '{"id":"%s","extract_uuid":"%s","salt":"%s","issued_at":"%s","hashing_scheme":"%s","hashes":[' \
        "$ID" "$EXTRACT_UUID" "$SALT" "$ISSUED_AT" "$HASHING_SCHEME"

    for h in "${REAL_HASHES[@]}"; do
        printf '"0x%s",' "$h"
    done

    for (( k=0; k<JOBS; k++ )); do
        chunk_file=$(printf '%s/chunk.%03d' "$TMPDIR_RUN" "$k")
        cat "$chunk_file"
    done

    printf '"0x%064x"' "$LAST_COUNTER"
    printf ']}'
} > "$OUT_PATH"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
ACTUAL_SIZE=$(file_size "$OUT_PATH")
EXPECTED_SIZE=$(( (HEADER_BYTES + 1) + 69 * TOTAL_HASHES ))

printf '\n'
printf 'wrote: %s\n' "$OUT_PATH"
printf 'size : %s bytes (target %s, expected %s)\n' "$ACTUAL_SIZE" "$TARGET_BYTES" "$EXPECTED_SIZE"
printf 'count: %s hashes (%s real + %s filler)\n' "$TOTAL_HASHES" "$REAL_COUNT" "$FILLER_COUNT"
printf 'id            : %s\n' "$ID"
printf 'extract_uuid  : %s\n' "$EXTRACT_UUID"
printf 'salt          : %s\n' "$SALT"
printf 'issued_at     : %s\n' "$ISSUED_AT"
printf 'hashing_scheme: %s\n' "$HASHING_SCHEME"
printf '\n'
printf 'real (address -> hash):\n'
for i in "${!REAL_ADDRESSES[@]}"; do
    printf '  %s -> 0x%s\n' "${REAL_ADDRESSES[$i]}" "${REAL_HASHES[$i]}"
done
