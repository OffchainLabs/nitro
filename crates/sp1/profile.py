#!/usr/bin/env python3
"""
Profile the SP1 zkVM pipeline for a given block input.

Usage:
    python3 profile.py --output-dir TARGET/sp1 --block-inputs-dir TARGET/sp1/block-inputs --block stylus

Phases measured:
  bootloading         — SP1 execution time only (excludes WASM→LLVM compilation)
  stylus_compilation  — one row per Stylus program compiled inside sp1-runner
  reexecution         — full block re-execution inside sp1-runner

All times are read from [PROFILE] log lines emitted by the binaries themselves.
"""

import argparse
import os
import re
import subprocess
import sys

# ---------------------------------------------------------------------------
# Log parsing
# ---------------------------------------------------------------------------

_PROFILE_RE = re.compile(r"\[PROFILE] (\w+): (.*)")
_KV_RE = re.compile(r"(\w+)=([^\s,]+)")


def parse_profile_lines(text: str) -> list[dict]:
    rows = []
    for line in text.splitlines():
        m = _PROFILE_RE.search(line)
        if not m:
            continue
        phase, kvs = m.group(1), m.group(2)
        row = {"phase": phase}
        for k, v in _KV_RE.findall(kvs):
            row[k] = v
        rows.append(row)
    return rows


# ---------------------------------------------------------------------------
# Running subprocesses
# ---------------------------------------------------------------------------

def run(label: str, cmd: list[str]) -> str:
    """Run cmd, print a progress label, return combined stderr+stdout."""
    print(f"  {label}...", flush=True)
    env = os.environ.copy()
    # Ensure INFO-level tracing is visible so [PROFILE] lines are emitted.
    env.setdefault("RUST_LOG", "info")
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    combined = result.stderr + result.stdout
    if result.returncode not in (0, 1):
        # exit code 1 is expected from sp1-builder (bootloading stops early)
        print(f"\nERROR: {label} exited with code {result.returncode}", file=sys.stderr)
        print(combined, file=sys.stderr)
        sys.exit(1)
    return combined


# ---------------------------------------------------------------------------
# Table formatting
# ---------------------------------------------------------------------------

def fmt_int(v: str | None) -> str:
    if v is None:
        return "—"
    try:
        return f"{int(v):,}"
    except ValueError:
        return v


def fmt_bytes(v: str | None) -> str:
    if v is None:
        return "—"
    try:
        n = int(v)
        return f"{n / 1024:.1f} KiB" if n >= 1024 else f"{n} B"
    except ValueError:
        return v


def fmt_secs(v: str | None) -> str:
    if v is None:
        return "—"
    try:
        return f"{float(v):.3f}s"
    except ValueError:
        return v


def print_table(rows: list[dict]) -> None:
    headers = ["Phase", "Wasm size", "SP1 cycles", "Time"]

    display = []
    for r in rows:
        display.append([
            r["label"],
            fmt_bytes(r.get("wasm_size")),
            fmt_int(r.get("cycles")),
            fmt_secs(r.get("time_secs")),
        ])

    col_widths = [
        max(len(headers[i]), max(len(d[i]) for d in display))
        for i in range(len(headers))
    ]

    # First column (phase label) is left-aligned; the rest are right-aligned.
    def fmt_cell(value: str, width: int, col: int) -> str:
        return value.ljust(width) if col == 0 else value.rjust(width)

    sep = "+-" + "-+-".join("-" * w for w in col_widths) + "-+"
    header_row = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, col_widths)) + " |"

    print()
    print(sep)
    print(header_row)
    print(sep)
    for d in display:
        print("| " + " | ".join(fmt_cell(c, w, i) for i, (c, w) in enumerate(zip(d, col_widths))) + " |")
    print(sep)
    print()


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--output-dir", required=True, help="Path to target/sp1")
    ap.add_argument("--block-inputs-dir", required=True, help="Path to target/sp1/block-inputs")
    ap.add_argument("--block", default="stylus", help="Block input name (without .json)")
    args = ap.parse_args()

    out = args.output_dir
    block_file = f"{args.block_inputs_dir}/{args.block}.json"

    print(f"\nProfiling block: {args.block}")

    print("\n[1/2] Running sp1-builder (WASM→LLVM compilation + SP1 bootloading):")
    boot_log = run(
        "sp1-builder",
        [
            "cargo", "run", "--release", "-p", "sp1-builder", "--",
            "--replay-wasm", f"{out}/replay.wasm",
            "--output-folder", out,
        ],
    )

    print("\n[2/2] Running sp1-runner (stylus compilation + reexecution):")
    run_log = run(
        "sp1-runner",
        [
            f"{out}/sp1-runner",
            "--program", f"{out}/dumped_replay_wasm.elf",
            "--stylus-compiler-program", f"{out}/stylus-compiler-program",
            "--block-file", block_file,
        ],
    )

    profile_rows = parse_profile_lines(boot_log) + parse_profile_lines(run_log)

    table: list[dict] = []
    stylus_count = 0
    for row in profile_rows:
        phase = row["phase"]
        if phase == "bootloading":
            table.append({"label": "bootloading", "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})
        elif phase == "stylus_compilation":
            stylus_count += 1
            table.append({"label": f"stylus_compilation [{stylus_count}]", "wasm_size": row.get("wasm_size"), "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})
        elif phase == "reexecution":
            table.append({"label": "reexecution", "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})

    if not table:
        print("\nNo [PROFILE] lines found. Make sure RUST_LOG is not suppressing INFO logs.", file=sys.stderr)
        sys.exit(1)

    print_table(table)


if __name__ == "__main__":
    main()
