#!/usr/bin/env python3
"""
Compute delta between two SP1 profile JSON snapshots and emit a
GitHub-flavoured markdown comment.

Usage:
    python3 profile_delta.py \
        --old old.json --new new.json \
        --output comment.md
"""
import argparse
import json
from itertools import zip_longest

MARKER = "<!-- sp1-profile-comment -->"
BLOCKS = ["transfer", "solidity", "stylus", "stylus_heavy", "mixed"]


def _parse(v) -> float | None:
    try:
        return float(v)
    except (TypeError, ValueError):
        return None


def _fmt_cycles(v: float | None) -> str:
    return "—" if v is None else f"{int(v):,}"


def _fmt_gas(v: float | None) -> str:
    return "—" if v is None else f"{int(v):,}"


def _fmt_bytes(v: float | None) -> str:
    if v is None:
        return "—"
    n = int(v)
    return f"{n / 1024:.1f} KiB" if n >= 1024 else f"{n} B"


def _fmt_secs(v: float | None) -> str:
    return "—" if v is None else f"{v:.3f} s"


FIELDS = [
    ("cycles",    "cycles",    _fmt_cycles),
    ("gas",       "gas",       _fmt_gas),
    ("wasm_size", "wasm size", _fmt_bytes),
    ("time_secs", "time",      _fmt_secs),
]


def _delta_cell(b: float | None, p: float | None, fmt) -> str:
    if b is None or p is None:
        return "—"
    d = p - b
    pct = (d / b * 100) if b != 0 else 0.0
    sign = "+" if d >= 0 else "-"
    return f"{sign}{fmt(abs(d))} ({sign}{abs(pct):.1f}%)"


def _md_table(rows: list[list[str]], headers: list[str]) -> str:
    widths = [max(len(h), *(len(r[i]) for r in rows)) for i, h in enumerate(headers)]
    sep = "| " + " | ".join("-" * w for w in widths) + " |"
    hdr = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, widths)) + " |"
    data = [
        "| " + " | ".join(
            c.rjust(w) if i >= 2 else c.ljust(w)
            for i, (c, w) in enumerate(zip(r, widths))
        ) + " |"
        for r in rows
    ]
    return "\n".join([hdr, sep] + data)


def _phase_rows(phase: str, b: dict, p: dict) -> list[list[str]]:
    rows = []
    for key, label, fmt in FIELDS:
        bv = _parse(b.get(key))
        pv = _parse(p.get(key))
        if bv is None and pv is None:
            continue
        rows.append([phase, label, fmt(bv), fmt(pv), _delta_cell(bv, pv, fmt)])
    return rows


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--old",    required=True, help="Old (base branch) profile JSON")
    ap.add_argument("--new",    required=True, help="New (PR branch) profile JSON")
    ap.add_argument("--output", required=True, help="Output markdown file")
    args = ap.parse_args()

    with open(args.old) as f:
        base = json.load(f)
    with open(getattr(args, "new")) as f:
        pr = json.load(f)

    headers = ["Phase", "Metric", "Base", "PR", "Delta"]
    lines = ["## SP1 Profile — Delta Report", ""]

    # Bootloading
    boot_rows = _phase_rows("bootloading", base.get("bootloading") or {}, pr.get("bootloading") or {})
    if boot_rows:
        lines += ["### Bootloading", "", _md_table(boot_rows, headers), ""]

    # Per-block sections
    for block in BLOCKS:
        b_blk = (base.get("blocks") or {}).get(block, {})
        p_blk = (pr.get("blocks") or {}).get(block, {})

        rows = []
        b_sc = b_blk.get("stylus_compilations") or []
        p_sc = p_blk.get("stylus_compilations") or []
        for i, (b_s, p_s) in enumerate(zip_longest(b_sc, p_sc, fillvalue={}), 1):
            rows.extend(_phase_rows(f"stylus_compilation[{i}]", b_s, p_s))
        rows.extend(_phase_rows("reexecution", b_blk.get("reexecution") or {}, p_blk.get("reexecution") or {}))

        if rows:
            lines += [f"### Block: {block}", "", _md_table(rows, headers), ""]

    lines.append(MARKER)

    with open(args.output, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"Delta comment written to {args.output}")


if __name__ == "__main__":
    main()
