#!/usr/bin/env python3
"""
Profile the SP1 zkVM pipeline across all block types.

Usage:
    python3 profile.py --output-dir TARGET/sp1 --block-inputs-dir TARGET/sp1/block-inputs

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

BLOCKS = ["transfer", "solidity", "stylus", "stylus_heavy", "mixed"]

# Sample 1 in every N cycles for the SP1 trace file.
# Lower = more detail, larger file; higher = coarser, smaller file.
TRACE_SAMPLE_RATE = 300

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

def run(label: str, cmd: list[str], extra_env: dict[str, str] | None = None) -> str:
    """Run cmd, print a progress label, return combined stderr+stdout."""
    print(f"  {label}...", flush=True)
    env = os.environ.copy()
    # Ensure INFO-level tracing is visible so [PROFILE] lines are emitted.
    env.setdefault("RUST_LOG", "info")
    if extra_env:
        env.update(extra_env)
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


# A table row is either a data dict or a section sentinel {"section": name}.

def print_table(rows: list[dict]) -> None:
    headers = ["Phase", "Wasm size", "SP1 cycles", "Time"]

    # Collect display cells for data rows only (to compute column widths).
    display: list[list[str] | str] = []  # str entries are section labels
    for r in rows:
        if "section" in r:
            display.append(r["section"])
        else:
            display.append([
                r["label"],
                fmt_bytes(r.get("wasm_size")),
                fmt_int(r.get("cycles")),
                fmt_secs(r.get("time_secs")),
            ])

    data_rows = [d for d in display if isinstance(d, list)]
    col_widths = [
        max(len(headers[i]), max(len(d[i]) for d in data_rows))
        for i in range(len(headers))
    ]

    total_inner = sum(col_widths) + 3 * (len(col_widths) - 1)  # widths + " | " separators

    def fmt_cell(value: str, width: int, col: int) -> str:
        return value.ljust(width) if col == 0 else value.rjust(width)

    sep      = "+-" + "-+-".join("-" * w for w in col_widths) + "-+"
    thick    = "+=" + "=+=".join("=" * w for w in col_widths) + "=+"
    hdr_row  = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, col_widths)) + " |"

    print()
    print(sep)
    print(hdr_row)
    print(sep)
    for item in display:
        if isinstance(item, str):
            # Section header row: block name centered across full table width.
            label = f" {item} "
            print(thick)
            print("| " + label.center(total_inner) + " |")
            print(sep)
        else:
            print("| " + " | ".join(fmt_cell(c, w, i) for i, (c, w) in enumerate(zip(item, col_widths))) + " |")
    print(sep)
    print()


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--output-dir", required=True, help="Path to target/sp1")
    ap.add_argument("--block-inputs-dir", required=True, help="Path to target/sp1/block-inputs")
    args = ap.parse_args()

    out = args.output_dir

    print("\n[1] Running sp1-builder (WASM→LLVM compilation + SP1 bootloading):")
    boot_log = run(
        "sp1-builder",
        [
            "cargo", "run", "--release", "-p", "sp1-builder", "--",
            "--replay-wasm", f"{out}/replay.wasm",
            "--output-folder", out,
        ],
    )

    table: list[dict] = []

    boot_rows = parse_profile_lines(boot_log)
    for row in boot_rows:
        if row["phase"] == "bootloading":
            table.append({"label": "bootloading", "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})

    print(f"\n[2] Running sp1-runner on {len(BLOCKS)} block types:")
    for i, block in enumerate(BLOCKS, 1):
        block_file = f"{args.block_inputs_dir}/{block}.json"
        trace_file = f"{out}/trace_{block}.json"
        run_log = run(
            f"sp1-runner [{block}]",
            [
                f"{out}/sp1-runner-profiling",
                "--program", f"{out}/dumped_replay_wasm.elf",
                "--stylus-compiler-program", f"{out}/stylus-compiler-program",
                "--block-file", block_file,
            ],
            extra_env={
                "TRACE_FILE": trace_file,
                "TRACE_SAMPLE_RATE": str(TRACE_SAMPLE_RATE),
            },
        )
        print(f"    trace -> {trace_file}")

        table.append({"section": block})
        stylus_count = 0
        for row in parse_profile_lines(run_log):
            phase = row["phase"]
            if phase == "stylus_compilation":
                stylus_count += 1
                table.append({"label": f"stylus_compilation [{stylus_count}]", "wasm_size": row.get("wasm_size"), "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})
            elif phase == "reexecution":
                table.append({"label": "reexecution", "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})

    data_rows = [r for r in table if "section" not in r]
    if not data_rows:
        print("\nNo [PROFILE] lines found. Make sure RUST_LOG is not suppressing INFO logs.", file=sys.stderr)
        sys.exit(1)

    print_table(table)


if __name__ == "__main__":
    main()
