#!/usr/bin/env bash
#
# Build wrapper for the CI go-tests job: npm intermittently hangs forever
# after a successful install (NIT-5420), so each build attempt is bounded and
# retried; make is incremental, so a retry resumes where the killed attempt
# left off. Before killing a hung attempt, diagnostics are captured from the
# hung processes (kernel-side state plus a Node diagnostic report listing the
# active libuv handles) into $HANG_REPORT_DIR, which CI uploads as an
# artifact, to support an upstream bug report to npm.
#
# The Node reports require the step to run with:
#   NODE_OPTIONS="--report-on-signal --report-signal=SIGUSR2 --report-directory=$HANG_REPORT_DIR"

set -uo pipefail
# Run background jobs in their own process group so a hung attempt (and every
# process it spawned) can be enumerated and killed as a unit.
set -m

ATTEMPTS="${ATTEMPTS:-3}"
ATTEMPT_TIMEOUT_SECS="${ATTEMPT_TIMEOUT_SECS:-600}"
HANG_REPORT_DIR="${HANG_REPORT_DIR:-hang-reports}"

mkdir -p "$HANG_REPORT_DIR"

# True while the process exists and is not a zombie (a finished child stays a
# zombie until reaped by `wait`, so `kill -0` alone would report it as alive).
proc_alive() {
    local state
    state=$(ps -o state= -p "$1" 2>/dev/null) || return 1
    [[ -n "$state" && $state != Z* ]]
}

capture_diagnostics() {
    local attempt=$1 pgid=$2 pid args
    {
        echo "### process tree"
        ps -ejfH
        echo
        echo "### sockets"
        ss -tanp 2>/dev/null
        while read -r pid args; do
            echo
            echo "### pid $pid: $args"
            echo "wchan:   $(cat "/proc/$pid/wchan" 2>/dev/null)"
            echo "syscall: $(cat "/proc/$pid/syscall" 2>/dev/null)"
            echo "open fds:"
            ls -l "/proc/$pid/fd" 2>/dev/null
            # Ask Node for a diagnostic report (JS stack + active libuv
            # handles — the conclusive data for a hang at exit). Only signal
            # processes armed with --report-on-signal: for anything else the
            # default reaction to SIGUSR2 is termination.
            if grep -qz -- --report-on-signal "/proc/$pid/environ" 2>/dev/null; then
                kill -USR2 "$pid" 2>/dev/null || true
            fi
        done < <(ps -eo pid=,pgid=,args= | awk -v pgid="$pgid" '$2 == pgid {print $1, substr($0, index($0, $3))}')
    } > "$HANG_REPORT_DIR/attempt-$attempt-processes.txt" 2>&1
    sleep 5 # let Node finish writing the reports triggered above
    echo "Diagnostics captured:"
    ls -l "$HANG_REPORT_DIR"
}

for attempt in $(seq 1 "$ATTEMPTS"); do
    make -j "$(nproc)" build test-go-deps &
    make_pid=$!

    deadline=$((SECONDS + ATTEMPT_TIMEOUT_SECS))
    while proc_alive "$make_pid" && ((SECONDS < deadline)); do
        sleep 5
    done

    if ! proc_alive "$make_pid"; then
        if wait "$make_pid"; then
            exit 0
        fi
        echo "::warning::Build attempt $attempt failed"
        continue
    fi

    echo "::warning::Build attempt $attempt hung after ${ATTEMPT_TIMEOUT_SECS}s; capturing diagnostics before retrying"
    capture_diagnostics "$attempt" "$make_pid"
    kill -TERM -- "-$make_pid" 2>/dev/null || true
    sleep 30
    kill -KILL -- "-$make_pid" 2>/dev/null || true
    wait "$make_pid" 2>/dev/null
done

echo "::error::Build failed after $ATTEMPTS attempts"
exit 1
