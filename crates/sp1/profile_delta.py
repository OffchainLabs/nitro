#!/usr/bin/env python3
"""
Compute delta between two profile JSON snapshots and emit a
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
BLOCKS = ["transfer", "solidity", "stylus", "stylus_heavy", "mixed", "signatures"]

# Crypto-related syscall name prefixes — sorted to the top of the syscall diff
# table (the migration we care about most affects SECP256K1_*).
_CRYPTO_SYSCALL_PREFIXES = ("SECP256K1", "KECCAK", "SHA256", "BN254", "BLS12381", "ED25519")


def _is_crypto_syscall(code: str) -> bool:
    return any(code.startswith(p) for p in _CRYPTO_SYSCALL_PREFIXES)


def _parse(v) -> float | None:
    try:
        return float(v)
    except (TypeError, ValueError):
        return None


def _fmt_int(v: float | None) -> str:
    return "—" if v is None else f"{int(v):,}"


def _fmt_ms(v: float | None) -> str:
    return "—" if v is None else f"{int(v):,} ms"


SP1_FIELDS = [
    ("cycles", "cycles", _fmt_int),
    ("gas",    "gas",    _fmt_int),
]


def _delta_cell(b: float | None, p: float | None, fmt) -> str:
    if b is None or p is None:
        return "—"
    d = p - b
    pct = (d / b * 100) if b != 0 else 0.0
    sign = "+" if d >= 0 else "-"
    return f"{sign}{fmt(abs(d))} ({sign}{abs(pct):.1f}%)"


def _md_table(rows: list[list[str]], headers: list[str], right_align_from: int = 2) -> str:
    widths = [max(len(h), *(len(r[i]) for r in rows)) for i, h in enumerate(headers)]
    sep = "| " + " | ".join("-" * w for w in widths) + " |"
    hdr = "| " + " | ".join(h.ljust(w) for h, w in zip(headers, widths)) + " |"
    data = [
        "| " + " | ".join(
            c.rjust(w) if i >= right_align_from else c.ljust(w)
            for i, (c, w) in enumerate(zip(r, widths))
        ) + " |"
        for r in rows
    ]
    return "\n".join([hdr, sep] + data)


def _sp1_rows(phase: str, b: dict, p: dict) -> list[list[str]]:
    rows = []
    for key, label, fmt in SP1_FIELDS:
        bv = _parse(b.get(key))
        pv = _parse(p.get(key))
        if bv is None and pv is None:
            continue
        rows.append([phase, label, fmt(bv), fmt(pv), _delta_cell(bv, pv, fmt)])
    return rows


def _syscall_rows(b: dict, p: dict) -> list[list[str]]:
    """Build [code, base, pr, delta] rows for syscalls present in either snapshot.

    Crypto-related codes (SECP256K1_*, KECCAK_*, etc.) are listed first.
    """
    b_sys = b.get("syscalls") or {}
    p_sys = p.get("syscalls") or {}
    codes = set(b_sys) | set(p_sys)
    if not codes:
        return []
    crypto = sorted(c for c in codes if _is_crypto_syscall(c))
    other = sorted(c for c in codes if not _is_crypto_syscall(c))
    rows = []
    for code in crypto + other:
        bv = _parse(b_sys.get(code))
        pv = _parse(p_sys.get(code))
        rows.append([code, _fmt_int(bv), _fmt_int(pv), _delta_cell(bv, pv, _fmt_int)])
    return rows


def _arbitrator_row(b_blk: dict, p_blk: dict) -> list[list[str]]:
    bv = _parse((b_blk.get("arbitrator") or {}).get("steps"))
    pv = _parse((p_blk.get("arbitrator") or {}).get("steps"))
    if bv is None and pv is None:
        return []
    return [["arbitrator", "WAVM steps", _fmt_int(bv), _fmt_int(pv), _delta_cell(bv, pv, _fmt_int)]]


def _jit_row(b_blk: dict, p_blk: dict) -> list[list[str]]:
    bv = _parse((b_blk.get("jit") or {}).get("time_ms"))
    pv = _parse((p_blk.get("jit") or {}).get("time_ms"))
    if bv is None and pv is None:
        return []
    return [["jit", "time", _fmt_ms(bv), _fmt_ms(pv), _delta_cell(bv, pv, _fmt_ms)]]


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
    boot_rows = _sp1_rows("bootloading", base.get("bootloading") or {}, pr.get("bootloading") or {})
    if boot_rows:
        lines += ["### Bootloading", "", _md_table(boot_rows, headers), ""]

    # Per-block sections
    for block in BLOCKS:
        b_blk = (base.get("blocks") or {}).get(block, {})
        p_blk = (pr.get("blocks") or {}).get(block, {})

        # Sort compilations by cycles so positional zip-pairing is stable
        # across runs (runner iteration order is HashMap-random).
        sort_key = lambda c: int(c.get("cycles") or 0)
        rows = []
        b_sc = sorted(b_blk.get("stylus_compilations") or [], key=sort_key)
        p_sc = sorted(p_blk.get("stylus_compilations") or [], key=sort_key)
        for i, (b_s, p_s) in enumerate(zip_longest(b_sc, p_sc, fillvalue={}), 1):
            rows.extend(_sp1_rows(f"stylus_compilation[{i}]", b_s, p_s))
        rows.extend(_sp1_rows("reexecution", b_blk.get("reexecution") or {}, p_blk.get("reexecution") or {}))
        rows.extend(_arbitrator_row(b_blk, p_blk))
        rows.extend(_jit_row(b_blk, p_blk))

        if rows:
            lines += [f"### Block: {block}", "", _md_table(rows, headers), ""]

        sys_rows = _syscall_rows(b_blk.get("reexecution") or {}, p_blk.get("reexecution") or {})
        if sys_rows:
            lines += [
                f"#### Block {block} — syscalls",
                "",
                _md_table(sys_rows, ["Syscall", "Base", "PR", "Delta"], right_align_from=1),
                "",
            ]

    lines.append(MARKER)

    with open(args.output, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"Delta comment written to {args.output}")


if __name__ == "__main__":
    main()
