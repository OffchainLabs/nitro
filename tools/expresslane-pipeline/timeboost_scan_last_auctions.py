#!/usr/bin/env python3
"""
timeboost_scan_last_auctions.py — Scan recent auction rounds for true timeboosted txs.

Reads a cached list of AuctionResolved block numbers (from /tmp/auction_blocks.json),
then scans the timeboost window after the last N auction events using
eth_getBlockReceipts and eth_getBlockByNumber to collect full tx info.

Usage:
  python3 timeboost_scan_last_auctions.py \
    --auction-file /tmp/auction_blocks.json \
    --last-n 10 --window 260 \
    --out output/timeboosted_true_<timestamp>.jsonl
"""

import argparse
import json
import time
from typing import Dict, List, Tuple

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

DEFAULT_RPC = "https://arb1.arbitrum.io/rpc"
SELECTORS = {
    "0xb2460c48": "unknown_b2460c48",
    "0x6bfd6286": "unknown_6bfd6286",
    "0x4a49d86e": "unknown_4a49d86e",
    "0x6bf6a42d": "checkCallback(uint256,uint256,bytes32)",
    "0x02001e01": "unknown_02001e01",
    "0x02003c01": "unknown_02003c01",
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
}

_rpc_id = 0

def rpc(url: str, method: str, params: list, timeout: int = 15, retries: int = 3):
    global _rpc_id
    _rpc_id += 1
    payload = {"jsonrpc": "2.0", "id": _rpc_id, "method": method, "params": params}
    last_err = None
    for attempt in range(retries):
        try:
            resp = _post(url, payload, timeout=timeout)
            if "error" in resp:
                raise RuntimeError(f"RPC error: {resp['error']}")
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


def scan_blocks(url: str, blocks: List[int], out_file, rate: float = 0.06, verbose: bool = True) -> Tuple[int,int]:
    total = 0
    blocks_boosted = 0
    for i, bn in enumerate(blocks):
        if i % 100 == 0 and verbose:
            print(f"  [scan] {i}/{len(blocks)} blocks, {total} boosted so far")
        try:
            receipts = rpc(url, "eth_getBlockReceipts", [hex(bn)], timeout=15)
        except Exception as e:
            print(f"  [warn] block {bn} receipts failed: {e}")
            time.sleep(1)
            continue
        if not receipts:
            time.sleep(rate)
            continue
        boosted = [r for r in receipts if r.get("timeboosted")]
        if not boosted:
            time.sleep(rate)
            continue
        blocks_boosted += 1
        blk = rpc(url, "eth_getBlockByNumber", [hex(bn), True], timeout=15)
        txs = {int(t.get("transactionIndex","0x0"),16): t for t in (blk.get("transactions",[]) if blk else [])}
        blk_ts = int(blk.get("timestamp","0x0"),16) if blk else 0
        for rcpt in boosted:
            tx_idx = int(rcpt.get("transactionIndex","0x0"),16)
            tx = txs.get(tx_idx, {})
            input_data = tx.get("input", "")
            selector, func = decode_selector(input_data)
            row = {
                "txHash": rcpt.get("transactionHash", tx.get("hash", "")),
                "block": bn,
                "timestamp": blk_ts,
                "tx_index": tx_idx,
                "from": (rcpt.get("from") or tx.get("from") or "").lower(),
                "to": (rcpt.get("to") or tx.get("to") or "").lower(),
                "value_wei": int(tx.get("value","0x0"),16) if isinstance(tx.get("value"),str) else 0,
                "selector": selector,
                "function_name": func,
                "input_len": len(input_data)//2 if input_data else 0,
                "timeboosted": True,
                "status": int(rcpt.get("status","0x1"),16),
                "gas_used": int(rcpt.get("gasUsed","0x0"),16),
                "gas_used_l1": int(rcpt.get("gasUsedForL1","0x0"),16),
                "l1_block": int(rcpt.get("l1BlockNumber","0x0"),16),
                "log_count": len(rcpt.get("logs", [])),
                "match_source": "receipt_timeboosted_flag",
            }
            out_file.write(json.dumps(row) + "\n")
            out_file.flush()
            total += 1
            if verbose:
                print(f"  [BOOSTED] block={bn} idx={tx_idx} to={row['to'][:18]} fn={func}")
        time.sleep(rate)
    return total, blocks_boosted


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--auction-file", required=True)
    p.add_argument("--last-n", type=int, default=10)
    p.add_argument("--window", type=int, default=260)
    p.add_argument("--out", required=True)
    p.add_argument("--rpc", default=DEFAULT_RPC)
    p.add_argument("--rate", type=float, default=0.06)
    args = p.parse_args()

    data = json.load(open(args.auction_file))
    auction_blocks = sorted(data.get("auction_blocks", []))
    if not auction_blocks:
        raise SystemExit("No auction blocks in file")

    recent = auction_blocks[-args.last_n:]
    print(f"Using last {len(recent)} auctions: {recent[0]}..{recent[-1]}")

    blocks = []
    for abn in recent:
        blocks.extend(range(abn + 1, abn + 1 + args.window))
    blocks = sorted(set(blocks))
    print(f"Scanning {len(blocks)} blocks (window={args.window})")

    with open(args.out, "w") as f:
        total, blocks_boosted = scan_blocks(args.rpc, blocks, f, rate=args.rate, verbose=True)

    print(f"\n✓ Done. {total} timeboosted txs across {blocks_boosted} blocks written to {args.out}")

if __name__ == "__main__":
    main()
