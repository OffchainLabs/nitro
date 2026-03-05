#!/usr/bin/env bash
# Generates a markdown table of top N contributors per codebase section (default 5).
# Usage: ./scripts/code-ownership.sh [-h|--help] [--since[=]DATE] [NUM_CONTRIBUTORS]

set -euo pipefail

# --- Constants & data ---

EXCLUDE=(
  'Lee Bousfield'
  'Rachel Franks'
  'Rachel Bousfield'
  'Tsahi Zidenberg'
  'Ed Felten'
  'dependabot[bot]'
  'github-actions'
  'fredlacs'
  'Jared Wasinger'
  'Matt Garnett'
  'Ricardo Catalinas'
  'kevaundray'
  'viktorking7'
  'Nodar Ambroladze'
  'Harry Kalodner'
)

# Section definitions: "label|description|space-separated paths..."
# Fields are pipe-delimited; paths are space-delimited (intentional word-splitting, see SC2086 disable).
SECTIONS=(
  "arbnode/|Consensus Node|arbnode/"
  "arbos/|L2 OS|arbos/"
  "arbstate/|Inbox Parsing|arbstate/"
  "consensus/|Consensus Interfaces + Wiring|consensus/ execution_consensus/"
  "execution/|Execution Node|execution/"
  "go-ethereum/|Vendored Geth Fork|go-ethereum gethhook/"
  "staker/|Block Validator + Challenge|staker/"
  "bold/|BoLD Challenge Protocol|bold/"
  "validator/|Validation Node (JIT + Native)|validator/"
  "cmd/replay/ + wavmio/|WASM Fraud Proof Replay|cmd/replay/ wavmio/"
  "crates/|Rust (Prover, JIT, Stylus)|crates/"
  "daprovider/|Data Availability (AnyTrust)|daprovider/"
  "timeboost/|Express Lane Sequencing|timeboost/"
  "precompiles/|Arbitrum Precompiles|precompiles/"
  "Feed (Broadcast + Relay)|WebSocket Feed, Clients, Relay|broadcaster/ broadcastclient/ broadcastclients/ wsbroadcastserver/ relay/ cmd/relay/"
  "pubsub/|Redis Pub/Sub Coordination|pubsub/"
  "util/|StopWaiter, RPCClient, Containers|util/ arbutil/"
  "solgen/|Solidity ABI -> Go Bindings|solgen/"
  "Build/CI/Tooling|Linters, Makefile, .github/|linters/ Makefile Dockerfile .github/ .dockerignore scripts/"
  "BatchPoster|arbnode/batch_poster.go|arbnode/batch_poster.go"
  "Sequencer|execution/gethexec/sequencer.go|execution/gethexec/sequencer.go"
  "DataPoster|arbnode/dataposter/|arbnode/dataposter/"
  "Inbox Reader/Tracker|arbnode/inbox_{reader,tracker}.go|arbnode/inbox_reader.go arbnode/inbox_tracker.go"
  "SeqCoordinator|arbnode/seq_coordinator.go|arbnode/seq_coordinator.go arbnode/seq_coordinator_test.go"
  "system_tests/|Integration Tests|system_tests/"
  "Delayed Sequencer|arbnode/delayed_sequencer.go|arbnode/delayed_sequencer.go"
  "blocks_reexecutor/|Block Re-execution|blocks_reexecutor/"
  "cmd/nitro/ + cmd/conf/|Node Entrypoint + Config|cmd/nitro/ cmd/conf/"
  "contracts/|Solidity Contracts|contracts/"
  "nitro-testnode/|Test Node Scripts|nitro-testnode/"
)

# --- Helper functions ---

# Normalize known duplicate git identities to canonical names.
# Runs before exclusion filtering, so canonical names must match EXCLUDE entries.
normalize_author() {
  sed \
    -e 's/^amsanghi$/Aman Sanghi/' \
    -e 's/^Nodar$/Nodar Ambroladze/' \
    -e 's/^Tristan-Wilson$/Tristan Wilson/' \
    -e 's/^ganeshvanahalli$/Ganesh Vanahalli/' \
    -e 's/^github-actions\[bot\]$/github-actions/'
}

# Strip leading count, keeping only the name (awk field splitting handles
# the variable-width leading whitespace that uniq -c produces).
format_contributor() {
  awk '{$1=""; sub(/^ /, ""); print}'
}

# Remove excluded authors from stdin (exit codes 0-1 from grep are treated
# as success since code 1 just means no lines matched).
filter_authors() {
  local rc=0
  grep -xvF -f <(printf '%s\n' "${EXCLUDE[@]}") || rc=$?
  [[ $rc -le 1 ]] || return "$rc"
}

# Print top N contributors by commit count for the given paths
top_n() {
  local n="$1"
  shift
  git log --format='%aN' --no-merges ${SINCE:+--since="$SINCE"} -- "$@" \
    | normalize_author \
    | filter_authors \
    | LC_ALL=C sort | uniq -c | sort -rn | head -"$n"
}

usage() { echo "Usage: $0 [-h|--help] [--since[=]DATE] [NUM_CONTRIBUTORS]" >&2; exit "${1:-1}"; }

# --- Core logic ---

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help) usage 0 ;;
      --since) [[ $# -ge 2 ]] || { echo "error: --since requires a value" >&2; exit 1; }; SINCE="$2"; shift 2 ;;
      --since=*) SINCE="${1#*=}"; [[ -n "$SINCE" ]] || { echo "error: --since requires a value" >&2; exit 1; }; shift ;;
      -*) echo "error: unknown flag: $1" >&2; exit 1 ;;
      *) [[ -z "${GOT_NUM:-}" ]] || { echo "error: unexpected argument: $1" >&2; exit 1; }; NUM="$1"; GOT_NUM=1; shift ;;
    esac
  done
  [[ "$NUM" =~ ^[1-9][0-9]*$ ]] || usage
}

# Process each section in a parallel subshell. Each writes its markdown row to a
# temp file and atomically renames it on success; results are reassembled in
# order after all jobs complete.
run_sections() {
  local section_idx=0
  for entry in "${SECTIONS[@]}"; do
    IFS='|' read -r label description paths <<< "$entry"
    section_labels+=("$label")
    (
      {
      # Disable globbing so word-split $paths are passed literally to git log.
      set -f
      # shellcheck disable=SC2086
      contributors=$(top_n "$NUM" $paths)

      if [[ -z "$contributors" ]]; then
        echo "warning: section '${label}' has zero contributors" > "$tmpdir/$section_idx.warn"
      fi

      cols=()
      if [[ -n "$contributors" ]]; then
        while IFS= read -r line; do
          cols+=("$line")
        done < <(echo "$contributors" | format_contributor)
      fi

      row="| ${label} | ${description} |"
      for ((i = 0; i < NUM; i++)); do
        row+=" ${cols[$i]:-} |"
      done
      echo "$row"
    # Atomic rename: final file only appears on success, so the reassembly
    # loop can distinguish success (file exists and non-empty) from failure.
    } > "$tmpdir/$section_idx.tmp" 2>"$tmpdir/$section_idx.err"
    mv "$tmpdir/$section_idx.tmp" "$tmpdir/$section_idx" 2>>"$tmpdir/$section_idx.err"
    ) &
    pids+=($!)
    ((++section_idx))
  done
}

wait_for_sections() {
  for pid in "${pids[@]}"; do
    wait "$pid" || true
  done
}

assemble_output() {
  failures=0
  outfile="$tmpdir/output.md"
  {
    echo "# Nitro Codebase Ownership"
    echo ""
    since_label="${SINCE:+ (since $SINCE)}"
    echo "Top ${NUM} contributors by commit count per section${since_label}. Generated on $(date -u +%Y-%m-%d)."
    echo ""

    # Header
    header="| Section | Description |"
    separator="| --- | --- |"
    for ((i = 1; i <= NUM; i++)); do
      header+=" #${i} |"
      separator+=" --- |"
    done
    echo "$header"
    echo "$separator"

    for ((i = 0; i < ${#SECTIONS[@]}; i++)); do
      [[ -s "$tmpdir/$i.warn" ]] && cat "$tmpdir/$i.warn" >&2
      if [[ -s "$tmpdir/$i" ]]; then
        cat "$tmpdir/$i"
      else
        echo "error: section '${section_labels[$i]}' produced no output" >&2
        [[ -s "$tmpdir/$i.err" ]] && cat "$tmpdir/$i.err" >&2
        ((++failures))
      fi
    done
  } > "$outfile"

  if [[ $failures -gt 0 ]]; then
    echo "error: $failures section(s) failed; output suppressed" >&2
    exit 1
  fi
  cat "$outfile"
}

# --- Main ---

toplevel=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "error: not in a git repository" >&2; exit 1; }
cd "$toplevel"

# Shared state across functions (declared here, set by parse_args / run_sections)
SINCE=""
NUM=5
section_labels=()
pids=()

parse_args "$@"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

run_sections
wait_for_sections
assemble_output
