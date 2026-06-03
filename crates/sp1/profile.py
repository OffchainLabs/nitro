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
import json
import os
import re
import subprocess
import sys

BLOCKS = ["transfer", "solidity", "stylus", "stylus_heavy", "mixed", "signatures"]

# Sample 1 in every N cycles for the SP1 trace file.
# Lower = more detail, larger file; higher = coarser, smaller file.
TRACE_SAMPLE_RATE = 300
JIT_RUNS = 20

# ---------------------------------------------------------------------------
# Log parsing
# ---------------------------------------------------------------------------

_PROFILE_RE = re.compile(r"\[PROFILE] (\w+): (.*)")
_KV_RE = re.compile(r"(\w+)=([^\s,]+)")

# Syscalls block emitted by sp1-runner Normal mode after `[PROFILE] reexecution: ...`.
# The runner uses bare `tracing::info!` lines (no `[PROFILE]` marker).
_SYSCALLS_HEADER_RE = re.compile(r"\bSyscalls:\s*$")
_SYSCALL_ROW_RE = re.compile(r"  ([A-Z][A-Z0-9_]+):\s+(\d+)\s*$")

_WAVM_STEPS_RE = re.compile(r"WAVM steps:\s*(\d+)")
_JIT_TIME_RE = re.compile(r"Completed in (\d+)ms")


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


def parse_syscalls(text: str) -> dict[str, int]:
    """Extract the syscalls map from a runner's combined output."""
    syscalls: dict[str, int] = {}
    in_syscalls = False
    for line in text.splitlines():
        if _PROFILE_RE.search(line):
            in_syscalls = False
            continue
        if _SYSCALLS_HEADER_RE.search(line):
            in_syscalls = True
            continue
        if in_syscalls:
            m = _SYSCALL_ROW_RE.search(line)
            if m:
                syscalls[m.group(1)] = int(m.group(2))
    return syscalls


# ---------------------------------------------------------------------------
# Running subprocesses
# ---------------------------------------------------------------------------

def run(label: str, cmd: list[str], extra_env: dict[str, str] | None = None,
        allowed_codes: tuple[int, ...] = (0, 1), quiet: bool = False) -> str:
    """Run cmd, print a progress label, return combined stderr+stdout.

    allowed_codes: exit codes that are not treated as errors.
    Default includes 1 because sp1-builder exits 1 on normal bootloading stop.
    Pass (0,) for binaries where any non-zero exit is a failure.
    Pass quiet=True to suppress the progress label (e.g. when the caller manages its own output).
    """
    if not quiet:
        print(f"  {label}...", flush=True)
    env = os.environ.copy()
    # Ensure INFO-level tracing is visible so [PROFILE] lines are emitted.
    env.setdefault("RUST_LOG", "info")
    if extra_env:
        env.update(extra_env)
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    combined = result.stderr + result.stdout
    if result.returncode not in allowed_codes:
        print(f"\nERROR: {label} exited with code {result.returncode}", file=sys.stderr)
        print(combined, file=sys.stderr)
        sys.exit(1)
    return combined


# ---------------------------------------------------------------------------
# Profilers  (one per validator)
# ---------------------------------------------------------------------------

def profile_sp1(out: str, block_inputs_dir: str) -> list[dict]:
    """Run SP1 bootloading + per-block reexecution; return table rows."""
    table: list[dict] = []

    print("\nRunning sp1-builder (WASM→LLVM compilation + SP1 bootloading):")
    boot_log = run(
        "sp1-builder",
        [
            "cargo", "run", "--release", "-p", "sp1-builder",
            "--features", "sp1-sdk/profiling,sp1-core-executor/profiling",
            "--",
            "--replay-wasm", f"{out}/replay.wasm",
            "--output-folder", out,
        ],
        extra_env={
            # TRACE_FILE must be set for the profiling feature to activate symbol
            # embedding in the dumped ELF. We use a throwaway path and a huge
            # sample rate so virtually no trace data is written.
            "TRACE_FILE": f"{out}/ignore_bootload_trace.json",
            "TRACE_SAMPLE_RATE": "1000000000",
        },
    )
    for row in parse_profile_lines(boot_log):
        if row["phase"] == "bootloading":
            table.append({"label": "bootloading", "cycles": row.get("cycles"),
                           "time_secs": row.get("time_secs")})

    print(f"\nRunning sp1-runner on {len(BLOCKS)} block types:")
    for block in BLOCKS:
        block_file = f"{block_inputs_dir}/{block}.json"
        trace_file = f"{out}/trace_{block}.json"
        run_log = run(
            f"sp1-runner [{block}]",
            [
                f"{out}/sp1-runner-profiling",
                "--program", f"{out}/dumped_replay_wasm.elf",
                "--stylus-compiler-program", f"{out}/stylus-compiler-program",
                "--block-file", block_file,
                "--mode", "normal",
            ],
            extra_env={
                "TRACE_FILE": trace_file,
                "TRACE_SAMPLE_RATE": str(TRACE_SAMPLE_RATE),
            },
        )
        print(f"    trace -> {trace_file}")

        table.append({"section": block})
        syscalls = parse_syscalls(run_log)
        stylus_count = 0
        for row in parse_profile_lines(run_log):
            if row["phase"] == "stylus_compilation":
                stylus_count += 1
                table.append({"label": f"stylus_compilation [{stylus_count}]",
                               "cycles": row.get("cycles"), "time_secs": row.get("time_secs")})
            elif row["phase"] == "reexecution":
                entry: dict = {"label": "reexecution", "cycles": row.get("cycles"),
                                "gas": row.get("gas"), "time_secs": row.get("time_secs")}
                if syscalls:
                    entry["syscalls"] = syscalls
                table.append(entry)

    return table


def profile_arbitrator(prover: str, machine: str, block_inputs_dir: str) -> dict[str, int | None]:
    """Run the arbitrator prover per block; return WAVM step counts."""
    print(f"\nRunning arbitrator prover on {len(BLOCKS)} block types:")
    steps: dict[str, int | None] = {}
    for block in BLOCKS:
        log = run(f"arbitrator [{block}]",
                  [prover, machine, "--json-inputs", f"{block_inputs_dir}/{block}.json",
                   "--count-steps", "--require-success"],
                  allowed_codes=(0,))
        m = _WAVM_STEPS_RE.search(log)
        steps[block] = int(m.group(1)) if m else None
    return steps


def profile_jit(jit: str, replay_wasm: str, block_inputs_dir: str) -> dict[str, int | None]:
    """Run the JIT validator per block (JIT_RUNS times); return min time in ms."""
    print(f"\nRunning JIT on {len(BLOCKS)} block types ({JIT_RUNS} runs each, reporting min):")
    times: dict[str, int | None] = {}
    for block in BLOCKS:
        block_file = f"{block_inputs_dir}/{block}.json"
        samples: list[int] = []
        for i in range(JIT_RUNS):
            print(f"  jit [{block}] {i + 1}/{JIT_RUNS}...", end="\r", flush=True)
            log = run(f"jit [{block}]",
                      [jit, "--debug", "--cranelift", "--binary", replay_wasm,
                       "json", f"--inputs={block_file}"],
                      allowed_codes=(0,), quiet=True)
            m = _JIT_TIME_RE.search(log)
            if m:
                samples.append(int(m.group(1)))
        result = f"min {min(samples)}ms" if samples else "—"
        print(f"  jit [{block}]: {result}".ljust(40))
    return times


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


def fmt_secs(v: str | None) -> str:
    if v is None:
        return "—"
    try:
        return f"{float(v):.3f}s"
    except ValueError:
        return v


def _print_simple_table(headers: list[str], rows: list[list[str]]) -> None:
    col_widths = [max(len(headers[i]), max(len(r[i]) for r in rows)) for i in range(len(headers))]
    sep = "+-" + "-+-".join("-" * w for w in col_widths) + "-+"
    hdr_row = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, col_widths)) + " |"
    print()
    print(sep)
    print(hdr_row)
    print(sep)
    for r in rows:
        print("| " + " | ".join(
            c.ljust(col_widths[i]) if i == 0 else c.rjust(col_widths[i])
            for i, c in enumerate(r)
        ) + " |")
    print(sep)
    print()


def print_sp1_table(rows: list[dict]) -> None:
    headers = ["Phase", "SP1 cycles", "Prover gas", "Time"]

    # Collect display cells for data rows only (to compute column widths).
    display: list[list[str] | str] = []  # str entries are section labels
    for r in rows:
        if "section" in r:
            display.append(r["section"])
        else:
            display.append([
                r["label"],
                fmt_int(r.get("cycles")),
                fmt_int(r.get("gas")),
                fmt_secs(r.get("time_secs")),
            ])

    data_rows = [d for d in display if isinstance(d, list)]
    col_widths = [
        max(len(headers[i]), max(len(d[i]) for d in data_rows))
        for i in range(len(headers))
    ]
    total_inner = sum(col_widths) + 3 * (len(col_widths) - 1)

    def fmt_cell(value: str, width: int, col: int) -> str:
        return value.ljust(width) if col == 0 else value.rjust(width)

    sep = "+-" + "-+-".join("-" * w for w in col_widths) + "-+"
    thick = "+=" + "=+=".join("=" * w for w in col_widths) + "=+"
    hdr_row = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, col_widths)) + " |"

    print()
    print(sep)
    print(hdr_row)
    print(sep)
    for item in display:
        if isinstance(item, str):
            print(thick)
            print("| " + f" {item} ".center(total_inner) + " |")
            print(sep)
        else:
            print("| " + " | ".join(fmt_cell(c, w, i) for i, (c, w) in enumerate(zip(item, col_widths))) + " |")
    print(sep)
    print()


def print_arb_jit_table(arb_steps: dict[str, int | None], jit_time_ms: dict[str, int | None]) -> None:
    rows = [
        [
            block,
            f"{arb_steps[block]:,}" if arb_steps.get(block) is not None else "—",
            f"{jit_time_ms[block] / 1000:.3f}s" if jit_time_ms.get(block) is not None else "—",
        ]
        for block in BLOCKS
    ]
    _print_simple_table(["Block", "WAVM steps", "JIT time"], rows)


# ---------------------------------------------------------------------------
# JSON output
# ---------------------------------------------------------------------------

_JSON_FIELDS = {"cycles", "gas", "syscalls"}


def write_json(table: list[dict], path: str,
               arb_steps: dict[str, int | None] | None = None,
               jit_time_ms: dict[str, int | None] | None = None) -> None:
    data: dict = {"bootloading": None, "blocks": {}}
    current = None
    for r in table:
        if "section" in r:
            current = r["section"]
            data["blocks"][current] = {"stylus_compilations": [], "reexecution": None}
        elif r.get("label") == "bootloading":
            data["bootloading"] = {k: v for k, v in r.items() if k in _JSON_FIELDS}
        elif r.get("label", "").startswith("stylus_compilation") and current:
            data["blocks"][current]["stylus_compilations"].append(
                {k: v for k, v in r.items() if k in _JSON_FIELDS}
            )
        elif r.get("label") == "reexecution" and current:
            data["blocks"][current]["reexecution"] = {k: v for k, v in r.items() if k in _JSON_FIELDS}
        else:
            print(f"write_json: unrecognised row, skipping: {r}", file=sys.stderr)
    for block, block_data in data["blocks"].items():
        if arb_steps and arb_steps.get(block) is not None:
            block_data["arbitrator"] = {"steps": arb_steps[block]}
        if jit_time_ms and jit_time_ms.get(block) is not None:
            block_data["jit"] = {"time_ms": jit_time_ms[block]}
    with open(path, "w") as f:
        json.dump(data, f, indent=2)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--output-dir", required=True, help="Path to target/sp1")
    ap.add_argument("--block-inputs-dir", required=True, help="Path to target/sp1/block-inputs")
    ap.add_argument("--json-output", help="Also write results as JSON to FILE")
    ap.add_argument("--prover", help="Path to the arbitrator-prover binary")
    ap.add_argument("--machine", help="Path to machine.v2.wavm.br")
    ap.add_argument("--jit", help="Path to the JIT binary")
    ap.add_argument("--replay-wasm", help="Path to replay.wasm for JIT (reference-types stripped)")
    ap.add_argument("--validators", nargs="+", choices=["sp1", "arbitrator", "jit"],
                    default=["sp1", "arbitrator", "jit"],
                    help="Which validators to profile (default: all)")
    args = ap.parse_args()

    table: list[dict] = []
    arb_steps: dict[str, int | None] = {}
    jit_time_ms: dict[str, int | None] = {}

    if "sp1" in args.validators:
        table = profile_sp1(args.output_dir, args.block_inputs_dir)
        if not any("section" not in r for r in table):
            print("\nNo [PROFILE] lines found. Make sure RUST_LOG is not suppressing INFO logs.",
                  file=sys.stderr)
            sys.exit(1)
        print_sp1_table(table)

    if "arbitrator" in args.validators and args.prover and args.machine:
        arb_steps = profile_arbitrator(args.prover, args.machine, args.block_inputs_dir)

    if "jit" in args.validators and args.jit and args.replay_wasm:
        jit_time_ms = profile_jit(args.jit, args.replay_wasm, args.block_inputs_dir)

    if arb_steps or jit_time_ms:
        print_arb_jit_table(arb_steps, jit_time_ms)

    if args.json_output:
        write_json(table, args.json_output, arb_steps or None, jit_time_ms or None)
        print(f"JSON written to {args.json_output}")


if __name__ == "__main__":
    main()
