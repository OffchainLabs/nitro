#!/usr/bin/env python3
"""
timeboost_receipt_extractor.py — True Timeboosted TX Extractor via eth_getBlockReceipts

Overcomes public RPC limitation where eth_getBlockByNumber tx objects lack the
`timeboosted` flag. Instead uses:

  - arb_getRawBlockMetadata  → bitmask-based fast pre-filter (picks blocks that
                               MAY have timeboosted txs based on Nitro metadata)
  - eth_getBlockReceipts     → full receipt set per block (timeboosted is present
                               in receipts on Arbitrum Nitro public RPC)
  - eth_getBlockByNumber     → tx details (from/to/input/value/index)

Additionally supports scanning around known AuctionResolved events for smarter
targeting (express lane is only useful right after a round starts).

Output: JSONL with one record per timeboosted transaction, enriched with:
  - tx details (from, to, selector, value, gas_used, status)
  - receipt fields (timeboosted, gasUsedForL1, l1BlockNumber)
  - block context (number, timestamp)
  - decoded selector name (where known)

Usage:
  python3 timeboost_receipt_extractor.py \\
    --from-block 433502500 --to-block 433503000 \\
    --out output/timeboosted_true.jsonl

  # Auto-detect from AuctionResolved events:
  python3 timeboost_receipt_extractor.py \\
    --auto-detect --range 200000 \\
    --out output/timeboosted_auto.jsonl
"""

import argparse
import json
import os
import sys
import time
from datetime import datetime
from typing import Any, Dict, List, Optional, Tuple

try:
    import requests as _requests
    def _post(url, payload, timeout=15):
        r = _requests.post(url, json=payload, timeout=timeout)
        r.raise_for_status()
        return r.json()
except ImportError:
    import urllib.request
    def _post(url, payload, timeout=15):  # type: ignore
        data = json.dumps(payload).encode()
        req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.loads(r.read())

# ─────────────────────────────────────────────────────────────────────────────
# Constants
# ─────────────────────────────────────────────────────────────────────────────

DEFAULT_RPC = os.environ.get("ARB_RPC", "https://arb1.arbitrum.io/rpc")
AUCTION_CONTRACT = "0x5fcb496a31b7ae91e7c9078ec662bd7a55cd3079"
AUCTION_RESOLVED_TOPIC = "0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47"
SET_ELC_TOPIC = "0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b"

# Known 4-byte selectors (union of auction + common DeFi)
SELECTORS = {
    # ExpressLaneAuction
    "0xb6b55f25": "deposit(uint256)",
    "0xb51d1d4f": "initiateWithdrawal()",
    "0xc5b6aa2f": "finalizeWithdrawal()",
    "0x6dc4fc4e": "resolveSingleBidAuction(...)",
    "0x447a709e": "resolveMultiBidAuction(...)",
    "0x007be2fe": "transferExpressLaneController(uint64,address)",
    "0xbef0ec74": "setTransferor(...)",
    "0x0d253fbe": "resolvedRounds()",
    # Common DeFi
    "0xa9059cbb": "transfer(address,uint256)",
    "0x23b872dd": "transferFrom(address,address,uint256)",
    "0x095ea7b3": "approve(address,uint256)",
    "0x38ed1739": "swapExactTokensForTokens(...)",
    "0x7ff36ab5": "swapExactETHForTokens(...)",
    "0x18cbafe5": "swapExactTokensForETH(...)",
    "0xd0e30db0": "deposit()",
    "0x2e1a7d4d": "withdraw(uint256)",
    "0xac9650d8": "multicall(bytes[])",
    "0x5ae401dc": "multicall(uint256,bytes[])",
    "0x12aa3caf": "swap(uint256,uint256,address[],bytes[],address)",
    "0xe449022e": "uniswapV3Swap(uint256,uint256,uint256[])",
    "0x0502b1c5": "unoswap(address,uint256,uint256,bytes32[])",
    "0xb2460c48": "unknown_b2460c48",
    "0x6bfd6286": "unknown_6bfd6286",
    "0x4a49d86e": "unknown_4a49d86e",
    "0x6bf6a42d": "checkCallback(uint256,uint256,bytes32)",  # chainlink
    "0x02001e01": "unknown_02001e01",
    "0x02003c01": "unknown_02003c01",
}

# ─────────────────────────────────────────────────────────────────────────────
# RPC helpers
# ─────────────────────────────────────────────────────────────────────────────

_rpc_id = 0

def rpc(url: str, method: str, params: list, timeout: int = 15, retries: int = 3) -> Any:
    global _rpc_id
    _rpc_id += 1
    payload = {"jsonrpc": "2.0", "id": _rpc_id, "method": method, "params": params}
    last_err = None
    for attempt in range(retries):
        try:
            resp = _post(url, payload, timeout=timeout)
            if "error" in resp:
                err = resp["error"]
                # Don't retry RPC-level errors (method not found, etc.)
                code = err.get("code", 0) if isinstance(err, dict) else 0
                if code in (-32601, -32602):
                    return None
                raise RuntimeError(f"RPC error: {err}")
            return resp.get("result")
        except Exception as e:
            last_err = e
            if attempt < retries - 1:
                time.sleep(1.0 * (attempt + 1))
    raise RuntimeError(f"RPC {method} failed after {retries} attempts: {last_err}")


def decode_selector(input_data: str) -> Tuple[str, str]:
    if not input_data or len(input_data) < 10:
        return "", "native_transfer_or_empty"
    sel = input_data[:10].lower()
    return sel, SELECTORS.get(sel, "unknown")


# ─────────────────────────────────────────────────────────────────────────────
# Phase 1: find candidate blocks via arb_getRawBlockMetadata
# ─────────────────────────────────────────────────────────────────────────────

def get_boosted_block_indices(url: str, from_block: int, to_block: int,
                              batch_size: int = 500, rate: float = 0.05) -> Dict[int, List[int]]:
    """
    Returns {block_number: [boosted_tx_indices]} for blocks that have
    non-zero timeboost bitmask in arb_getRawBlockMetadata.
    """
    result: Dict[int, List[int]] = {}
    total = to_block - from_block + 1
    processed = 0

    for start in range(from_block, to_block + 1, batch_size):
        end = min(start + batch_size - 1, to_block)
        try:
            meta = rpc(url, "arb_getRawBlockMetadata", [hex(start), hex(end)], timeout=20)
        except Exception as e:
            print(f"  [meta] {start}-{end} error: {e}", file=sys.stderr)
            time.sleep(1)
            continue

        if not meta:
            processed += (end - start + 1)
            time.sleep(rate)
            continue

        for entry in meta:
            raw = entry.get("rawMetadata", "0x")
            bn = entry.get("blockNumber", 0)
            if isinstance(bn, str):
                bn = int(bn, 16) if bn.startswith("0x") else int(bn)

            rb = bytes.fromhex(raw[2:]) if len(raw) > 2 else b""
            if len(rb) <= 1:
                continue

            boosted_idx: List[int] = []
            for i, byte in enumerate(rb[1:]):
                for bit in range(8):
                    if byte & (1 << bit):
                        boosted_idx.append(i * 8 + bit)

            if boosted_idx:
                result[bn] = boosted_idx

        processed += (end - start + 1)
        pct = processed / total * 100
        if processed % (batch_size * 5) == 0 or processed == total:
            print(f"  [meta-scan] {processed}/{total} ({pct:.1f}%) blocks, "
                  f"{len(result)} with boosted markers", flush=True)
        time.sleep(rate)

    return result


# ─────────────────────────────────────────────────────────────────────────────
# Phase 2: receipt scan (the authoritative approach)
# ─────────────────────────────────────────────────────────────────────────────

def scan_blocks_receipts(
    url: str,
    block_list: List[int],
    out_file,
    rate: float = 0.06,
    verbose: bool = True,
) -> Tuple[int, int]:
    """
    For each block in block_list, fetch all receipts via eth_getBlockReceipts.
    Emit JSONL rows for timeboosted=True receipts, enriched with tx details.

    Returns (total_timeboosted, total_blocks_with_boosted).
    """
    total_boosted = 0
    blocks_with_boosted = 0
    block_cache: Dict[int, Dict] = {}

    for i, bn in enumerate(block_list):
        if i % 100 == 0 and verbose:
            print(f"  [receipt-scan] {i}/{len(block_list)} blocks, {total_boosted} boosted so far", flush=True)

        try:
            receipts = rpc(url, "eth_getBlockReceipts", [hex(bn)], timeout=15)
        except Exception as e:
            print(f"  [warn] eth_getBlockReceipts({bn}) failed: {e}", file=sys.stderr)
            time.sleep(1)
            continue

        if not receipts:
            time.sleep(rate)
            continue

        # Filter to timeboosted ones
        boosted_rcpts = [r for r in receipts if r.get("timeboosted")]
        if not boosted_rcpts:
            time.sleep(rate)
            continue

        # Fetch full block with tx details (needed for input/selector/value)
        if bn not in block_cache:
            try:
                blk = rpc(url, "eth_getBlockByNumber", [hex(bn), True], timeout=15)
                block_cache[bn] = blk or {}
            except Exception as e:
                print(f"  [warn] eth_getBlockByNumber({bn}) failed: {e}", file=sys.stderr)
                block_cache[bn] = {}
            time.sleep(rate * 0.5)

        blk = block_cache[bn]
        blk_ts = blk.get("timestamp", "0x0")
        if isinstance(blk_ts, str):
            blk_ts = int(blk_ts, 16)

        # Build tx index map from block
        tx_by_index: Dict[int, Dict] = {}
        for tx in blk.get("transactions", []):
            tx_idx = tx.get("transactionIndex", "0x0")
            if isinstance(tx_idx, str):
                tx_idx = int(tx_idx, 16)
            tx_by_index[tx_idx] = tx

        blocks_with_boosted += 1
        for rcpt in boosted_rcpts:
            tx_idx_raw = rcpt.get("transactionIndex", "0x0")
            tx_idx = int(tx_idx_raw, 16) if isinstance(tx_idx_raw, str) else tx_idx_raw
            tx = tx_by_index.get(tx_idx, {})

            gas_used_raw = rcpt.get("gasUsed", "0x0")
            gas_used = int(gas_used_raw, 16) if isinstance(gas_used_raw, str) else gas_used_raw
            gas_used_l1_raw = rcpt.get("gasUsedForL1", "0x0")
            gas_used_l1 = int(gas_used_l1_raw, 16) if isinstance(gas_used_l1_raw, str) else 0
            l1_block_raw = rcpt.get("l1BlockNumber", "0x0")
            l1_block = int(l1_block_raw, 16) if isinstance(l1_block_raw, str) else 0
            status_raw = rcpt.get("status", "0x1")
            status = int(status_raw, 16) if isinstance(status_raw, str) else status_raw
            value_raw = tx.get("value", "0x0")
            value_wei = int(value_raw, 16) if isinstance(value_raw, str) else 0
            block_num_raw = rcpt.get("blockNumber", hex(bn))
            block_num = int(block_num_raw, 16) if isinstance(block_num_raw, str) else bn

            input_data = tx.get("input", "")
            selector, func_name = decode_selector(input_data)

            # Decode relevant receipt logs
            receipt_logs = []
            for log in rcpt.get("logs", []):
                topics = log.get("topics", [])
                if not topics:
                    continue
                log_entry = {
                    "address": log.get("address", "").lower(),
                    "logIndex": int(log.get("logIndex", "0x0"), 16),
                    "topic0": topics[0].lower() if topics else "",
                    "data_len": len(log.get("data", "0x")) // 2,
                }
                receipt_logs.append(log_entry)

            row = {
                "txHash": rcpt.get("transactionHash", tx.get("hash", "")),
                "block": block_num,
                "timestamp": blk_ts,
                "tx_index": tx_idx,
                "from": rcpt.get("from", tx.get("from", "")).lower(),
                "to": (rcpt.get("to") or tx.get("to") or "").lower(),
                "value_wei": value_wei,
                "selector": selector,
                "function_name": func_name,
                "input_len": len(input_data) // 2 if input_data else 0,
                "timeboosted": True,
                "status": status,
                "gas_used": gas_used,
                "gas_used_l1": gas_used_l1,
                "l1_block": l1_block,
                "contract_address": rcpt.get("contractAddress"),
                "log_count": len(rcpt.get("logs", [])),
                "receipt_logs": receipt_logs,
                "match_source": "receipt_timeboosted_flag",
                "effective_gas_price": rcpt.get("effectiveGasPrice", ""),
            }
            out_file.write(json.dumps(row) + "\n")
            out_file.flush()
            total_boosted += 1

            if verbose:
                print(f"  [BOOSTED] block={block_num} idx={tx_idx} "
                      f"from={row['from'][:14]}.. to={row['to'][:14]}.. "
                      f"fn={func_name} gas={gas_used}", flush=True)

        time.sleep(rate)

    return total_boosted, blocks_with_boosted


# ─────────────────────────────────────────────────────────────────────────────
# Strategy: auto-detect from AuctionResolved events
# ─────────────────────────────────────────────────────────────────────────────

def get_auction_resolved_blocks(url: str, from_block: int, to_block: int) -> List[int]:
    """Get L2 blocks where AuctionResolved was emitted."""
    try:
        logs = rpc(url, "eth_getLogs", [{
            "fromBlock": hex(from_block),
            "toBlock": hex(to_block),
            "address": AUCTION_CONTRACT,
            "topics": [AUCTION_RESOLVED_TOPIC],
        }], timeout=30)
        return [int(l["blockNumber"], 16) for l in (logs or [])]
    except Exception as e:
        print(f"  [warn] eth_getLogs for AuctionResolved failed: {e}", file=sys.stderr)
        return []


def blocks_in_windows(auction_blocks: List[int], window: int = 300) -> List[int]:
    """
    Generate the list of L2 blocks to scan based on auction resolution windows.
    Each auction round is ~59s = ~236 Arb blocks at 0.25s/block.
    We scan 'window' blocks after each auction resolution.
    """
    candidates: set = set()
    for abn in auction_blocks:
        for offset in range(1, window + 1):
            candidates.add(abn + offset)
    return sorted(candidates)


# ─────────────────────────────────────────────────────────────────────────────
# Combined metadata + receipt scan
# ─────────────────────────────────────────────────────────────────────────────

def scan_metadata_then_receipts(
    url: str,
    from_block: int,
    to_block: int,
    out_file,
    meta_batch: int = 500,
    rate: float = 0.06,
) -> Tuple[int, int, int]:
    """
    Two-phase:
      1. arb_getRawBlockMetadata for the full range → find candidate blocks
      2. eth_getBlockReceipts for candidate blocks → confirm timeboosted

    Returns (total_timeboosted, blocks_with_boosted, candidate_blocks_count).
    """
    total_range = to_block - from_block + 1
    print(f"[phase1] Scanning metadata for {total_range} blocks ({from_block}–{to_block})")

    boosted_map = get_boosted_block_indices(url, from_block, to_block,
                                            batch_size=meta_batch, rate=rate * 0.5)

    if not boosted_map:
        print("[phase1] No blocks with boosted metadata markers found. "
              "Falling back to full receipt scan of sampled blocks.")
        # Fallback: scan every N-th block
        sample = list(range(from_block, to_block + 1, 10))
        candidates = sample
    else:
        candidates = sorted(boosted_map.keys())
        print(f"[phase1] {len(candidates)} candidate blocks with boosted markers")

    print(f"[phase2] Receipt scanning {len(candidates)} candidate blocks")
    total_boosted, blocks_boosted = scan_blocks_receipts(
        url, candidates, out_file, rate=rate, verbose=True
    )
    return total_boosted, blocks_boosted, len(candidates)


# ─────────────────────────────────────────────────────────────────────────────
# Full receipt scan (no metadata pre-filter)
# ─────────────────────────────────────────────────────────────────────────────

def scan_full_receipts(
    url: str,
    from_block: int,
    to_block: int,
    out_file,
    rate: float = 0.06,
    chunk_size: int = 50,
) -> Tuple[int, int]:
    """Scan ALL blocks in range via eth_getBlockReceipts."""
    total_range = to_block - from_block + 1
    print(f"[full-scan] Receipt scanning {total_range} blocks ({from_block}–{to_block})")
    blocks = list(range(from_block, to_block + 1))
    return scan_blocks_receipts(url, blocks, out_file, rate=rate, verbose=True)


# ─────────────────────────────────────────────────────────────────────────────
# Auto-detect mode: target blocks after AuctionResolved events
# ─────────────────────────────────────────────────────────────────────────────

def scan_auto_detect(
    url: str,
    lookback: int,
    out_file,
    window: int = 300,
    rate: float = 0.06,
) -> Tuple[int, int, int]:
    """
    Find AuctionResolved events in the last `lookback` blocks, then scan
    `window` blocks after each one for timeboosted txs.
    """
    latest = int(rpc(url, "eth_blockNumber", []), 16)
    from_block = latest - lookback
    print(f"[auto] Latest block: {latest}")
    print(f"[auto] Scanning for AuctionResolved events in {from_block}–{latest}")

    auction_blocks = get_auction_resolved_blocks(url, from_block, latest)
    print(f"[auto] Found {len(auction_blocks)} AuctionResolved events")

    if not auction_blocks:
        print("[auto] No auction events found; scanning last 500 blocks fully")
        blocks = list(range(latest - 500, latest + 1))
    else:
        # Scan window blocks after each auction
        blocks = blocks_in_windows(auction_blocks, window=window)
        # Clamp to known range
        blocks = [b for b in blocks if from_block <= b <= latest]
        print(f"[auto] Scanning {len(blocks)} blocks in auction windows "
              f"({window} blocks per auction)")

    total_boosted, blocks_boosted = scan_blocks_receipts(
        url, blocks, out_file, rate=rate, verbose=True
    )
    return total_boosted, blocks_boosted, len(auction_blocks)


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Extract true timeboosted txs via receipt scan on Arbitrum One",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("--rpc", default=DEFAULT_RPC, help="Arbitrum RPC URL")
    parser.add_argument("--from-block", type=int, default=0)
    parser.add_argument("--to-block", type=int, default=0)
    parser.add_argument("--out", required=True, help="Output JSONL path")
    parser.add_argument("--rate", type=float, default=0.06, help="Sleep between calls (s)")

    # Mode selection
    mode_group = parser.add_mutually_exclusive_group(required=True)
    mode_group.add_argument("--full-scan", action="store_true",
                            help="Scan every block in --from-block to --to-block via receipts")
    mode_group.add_argument("--meta-scan", action="store_true",
                            help="Use arb_getRawBlockMetadata pre-filter + receipt confirmation")
    mode_group.add_argument("--auto-detect", action="store_true",
                            help="Find AuctionResolved events, then scan windows after each")
    mode_group.add_argument("--auction-windows", action="store_true",
                            help="Scan N blocks after each block in --from-block..--to-block "
                                 "that contains an AuctionResolved event")

    parser.add_argument("--range", type=int, default=200000,
                        help="For --auto-detect: look back N blocks from latest (default: 200000)")
    parser.add_argument("--window", type=int, default=300,
                        help="Blocks to scan after each auction resolution (default: 300)")
    parser.add_argument("--meta-batch", type=int, default=500,
                        help="Batch size for arb_getRawBlockMetadata (default: 500)")
    parser.add_argument("--chunk-size", type=int, default=50,
                        help="Chunk size for receipt scanning progress display")

    args = parser.parse_args()

    os.makedirs(os.path.dirname(os.path.abspath(args.out)), exist_ok=True)

    ts = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
    print(f"[{ts}] timeboost_receipt_extractor.py starting")
    print(f"  RPC: {args.rpc}")
    print(f"  Output: {args.out}")
    print(f"  Mode: {'full-scan' if args.full_scan else 'meta-scan' if args.meta_scan else 'auto-detect' if args.auto_detect else 'auction-windows'}")

    t_start = time.time()

    with open(args.out, "w") as out_file:
        if args.full_scan:
            if not args.from_block or not args.to_block:
                parser.error("--from-block and --to-block required for --full-scan")
            total, blocks_boosted = scan_full_receipts(
                args.rpc, args.from_block, args.to_block, out_file,
                rate=args.rate,
            )
            print(f"\n✓ Full scan complete. {total} timeboosted txs across "
                  f"{blocks_boosted} blocks.")

        elif args.meta_scan:
            if not args.from_block or not args.to_block:
                parser.error("--from-block and --to-block required for --meta-scan")
            total, blocks_boosted, candidates = scan_metadata_then_receipts(
                args.rpc, args.from_block, args.to_block, out_file,
                meta_batch=args.meta_batch, rate=args.rate,
            )
            print(f"\n✓ Meta+receipt scan complete. {total} timeboosted txs in "
                  f"{blocks_boosted} blocks (from {candidates} candidates).")

        elif args.auto_detect:
            total, blocks_boosted, n_auctions = scan_auto_detect(
                args.rpc, lookback=args.range, out_file=out_file,
                window=args.window, rate=args.rate,
            )
            print(f"\n✓ Auto-detect complete. {total} timeboosted txs in "
                  f"{blocks_boosted} blocks ({n_auctions} auction events detected).")

        elif args.auction_windows:
            if not args.from_block or not args.to_block:
                parser.error("--from-block and --to-block required for --auction-windows")
            auction_blocks = get_auction_resolved_blocks(
                args.rpc, args.from_block, args.to_block
            )
            print(f"[auction-windows] {len(auction_blocks)} AuctionResolved events in range")
            blocks = blocks_in_windows(auction_blocks, window=args.window)
            blocks = [b for b in blocks if args.from_block <= b <= args.to_block + args.window]
            total, blocks_boosted = scan_blocks_receipts(
                args.rpc, blocks, out_file, rate=args.rate, verbose=True
            )
            print(f"\n✓ Auction-windows scan complete. {total} timeboosted txs in "
                  f"{blocks_boosted} blocks.")

    elapsed = time.time() - t_start
    print(f"  Elapsed: {elapsed:.1f}s")
    print(f"  Output: {args.out}")

    # Quick stats from output
    try:
        rows = []
        with open(args.out) as f:
            for line in f:
                line = line.strip()
                if line:
                    rows.append(json.loads(line))
        if rows:
            senders = {}
            contracts = {}
            selectors = {}
            for row in rows:
                senders[row.get("from", "")] = senders.get(row.get("from", ""), 0) + 1
                contracts[row.get("to", "")] = contracts.get(row.get("to", ""), 0) + 1
                sel = row.get("selector", "") or "(native)"
                selectors[sel] = selectors.get(sel, 0) + 1

            print(f"\n=== Quick Stats ===")
            print(f"Total timeboosted txs: {len(rows)}")
            print(f"Top 5 senders: {sorted(senders.items(), key=lambda x: -x[1])[:5]}")
            print(f"Top 5 contracts: {sorted(contracts.items(), key=lambda x: -x[1])[:5]}")
            print(f"Top selectors: {sorted(selectors.items(), key=lambda x: -x[1])[:10]}")
    except Exception as e:
        print(f"(stats failed: {e})")


if __name__ == "__main__":
    main()
