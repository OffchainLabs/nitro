#!/usr/bin/env python3
"""
expresslane_filter.py — Arbitrum Timeboost / Express Lane TX Filter Pipeline
===========================================================================

Ingests a block range (or a pre-fetched JSON dataset) via the Ethereum JSON-RPC
and emits every transaction that touched the ExpressLaneAuction contract (Layer 1 /
Arbitrum parent chain side) plus every *timeboosted* transaction on the Arbitrum
child-chain side.

Two data sources are supported:
  1. PARENT CHAIN (L1-side, Ethereum Mainnet / Sepolia)
     - Transactions TO the ExpressLaneAuction contract
     - Topics from AuctionResolved / SetExpressLaneController events (log scan)
  2. CHILD CHAIN (L2-side, Arbitrum One / Arbitrum Sepolia)
     - Blocks where tx.timeboosted == true  (via Nitro's `arb_getBlockByHash` / `eth_getBlockByNumber`)
     - OR: blocks where the sequencer tagged a tx with the timeboost flag in the extra field

Outputs structured JSONL (one JSON object per candidate transaction).

Usage
-----
  # Scan L1 auction contract calls in block range:
  python3 expresslane_filter.py \\
    --mode l1-auction \\
    --rpc $ETH_RPC \\
    --contract 0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079 \\
    --from-block 20000000 --to-block 20001000 \\
    --out sample_output.jsonl

  # Scan L2 timeboosted txs:
  python3 expresslane_filter.py \\
    --mode l2-timeboosted \\
    --rpc $ARB_RPC \\
    --from-block 250000000 --to-block 250000100 \\
    --out sample_output.jsonl

  # Read from pre-fetched JSON dataset:
  python3 expresslane_filter.py \\
    --mode dataset \\
    --dataset txs.json \\
    --out sample_output.jsonl

Environment
-----------
  ETH_RPC   - Ethereum Mainnet RPC URL (for --mode l1-auction)
  ARB_RPC   - Arbitrum One RPC URL     (for --mode l2-timeboosted)
  ETHERSCAN_KEY - optional, for receipt fetches via Etherscan fallback

Dependencies: requests (stdlib-compatible fallback also provided)
"""

import argparse
import json
import os
import sys
import time
from typing import Any, Dict, List, Optional

try:
    import requests as _requests
    def http_post(url: str, payload: dict, timeout: int = 30) -> dict:
        resp = _requests.post(url, json=payload, timeout=timeout)
        resp.raise_for_status()
        return resp.json()
except ImportError:
    import urllib.request, urllib.error
    def http_post(url: str, payload: dict, timeout: int = 30) -> dict:  # type: ignore
        data = json.dumps(payload).encode()
        req = urllib.request.Request(url, data=data,
                                     headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.loads(r.read())


# ─────────────────────────────────────────────────────────────────────────────
# CONSTANTS — derived from IExpressLaneAuction.sol + keccak256 selectors
# ─────────────────────────────────────────────────────────────────────────────

#: Arbitrum One mainnet deployment (proxy)
EXPR_LANE_AUCTION_MAINNET = "0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079"

# Function 4-byte selectors  (keccak256(sig)[:4])
SELECTORS: Dict[str, str] = {
    "0xb6b55f25": "deposit(uint256)",
    "0xb51d1d4f": "initiateWithdrawal()",
    "0xc5b6aa2f": "finalizeWithdrawal()",
    "0x6ad72517": "flushBeneficiaryBalance()",
    "0x6dc4fc4e": "resolveSingleBidAuction((address,uint256,bytes))",
    "0x447a709e": "resolveMultiBidAuction((address,uint256,bytes),(address,uint256,bytes))",
    "0xbef0ec74": "setTransferor((address,uint64))",
    "0x007be2fe": "transferExpressLaneController(uint64,address)",
    "0xf698da25": "domainSeparator()",
    "0x04c584ad": "getBidHash(uint64,address,uint256)",
    "0x70a08231": "balanceOf(address)",
    "0x5633c337": "balanceOfAtRound(address,uint64)",
    "0x02b62938": "withdrawableBalance(address)",
    "0x6e8cace5": "withdrawableBalanceAtRound(address,uint64)",
    "0x0d253fbe": "resolvedRounds()",
    "0xce9c7c0d": "setReservePrice(uint256)",
    "0xe4d20c1d": "setMinReservePrice(uint256)",
    "0x1c31f710": "setBeneficiary(address)",
    "0xfed87be8": "setRoundTimingInfo((int64,uint64,uint64,uint64))",
    "0x7b617f94": "roundTimestamps(uint64)",
}

# Event topic0 hashes (keccak256(sig))
EVENT_TOPICS: Dict[str, str] = {
    "0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c":
        "Deposit(address,uint256)",
    "0x31f69201fab7912e3ec9850e3ab705964bf46d9d4276bdcbb6d05e965e5f5401":
        "WithdrawalInitiated(address,uint256,uint256)",
    "0x9e5c4f9f4e46b8629d3dda85f43a69194f50254404a72dc62b9e932d9c94eda8":
        "WithdrawalFinalized(address,uint256)",
    "0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47":
        "AuctionResolved(bool,uint64,address,address,uint256,uint256,uint64,uint64)",
    "0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b":
        "SetExpressLaneController(uint64,address,address,address,uint64,uint64)",
    "0xf6d28df235d9fa45a42d45dbb7c4f4ac76edb51e528f09f25a0650d32b8b33c0":
        "SetTransferor(address,address,uint64)",
    "0x5848068f11aa3ba9fe3fc33c5f9f2a3cd1aed67986b85b5e0cedc67dbe96f0f0":
        "SetMinReservePrice(uint256,uint256)",
    "0x9725e37e079c5bda6009a8f54d86265849f30acf61c630f9e1ac91e67de98794":
        "SetReservePrice(uint256,uint256)",
    "0x8a0149b2f3ddf2c9ee85738165131d82babbb938f749321d59f75750afa7f4e6":
        "SetBeneficiary(address,address)",
}

# High-value selectors for auction mechanics (subset — used for priority flagging)
AUCTION_SELECTORS = {
    "0x6dc4fc4e",  # resolveSingleBidAuction
    "0x447a709e",  # resolveMultiBidAuction
    "0x007be2fe",  # transferExpressLaneController
}

# Selectors that indicate lifecycle / balance operations
LIFECYCLE_SELECTORS = {
    "0xb6b55f25",  # deposit
    "0xb51d1d4f",  # initiateWithdrawal
    "0xc5b6aa2f",  # finalizeWithdrawal
    "0xbef0ec74",  # setTransferor
}


# ─────────────────────────────────────────────────────────────────────────────
# RPC helpers
# ─────────────────────────────────────────────────────────────────────────────

_rpc_id = 0

def rpc(url: str, method: str, params: list, retries: int = 3) -> Any:
    global _rpc_id
    _rpc_id += 1
    payload = {"jsonrpc": "2.0", "id": _rpc_id, "method": method, "params": params}
    for attempt in range(retries):
        try:
            resp = http_post(url, payload)
            if "error" in resp:
                raise RuntimeError(f"RPC error: {resp['error']}")
            return resp.get("result")
        except Exception as e:
            if attempt == retries - 1:
                raise
            time.sleep(1.5 ** attempt)


def hex_to_int(h: Optional[str]) -> int:
    if not h:
        return 0
    return int(h, 16)


# ─────────────────────────────────────────────────────────────────────────────
# Decoding helpers
# ─────────────────────────────────────────────────────────────────────────────

def decode_selector(input_data: str) -> tuple[str, str]:
    """Return (selector_hex, human_name). selector_hex is '' for empty input."""
    if not input_data or len(input_data) < 10:
        return "", "native_transfer_or_empty"
    sel = input_data[:10].lower()
    return sel, SELECTORS.get(sel, "unknown")


def decode_high_level_fields(input_data: str, selector: str) -> Dict[str, Any]:
    """
    Best-effort ABI decode of the most important functions.
    Returns decoded fields where feasible, raw hex otherwise.
    Full ABI decode requires eth_abi; we handle the simple cases manually.
    """
    fields: Dict[str, Any] = {}
    if not input_data or len(input_data) < 10:
        return fields

    calldata = input_data[10:]  # strip selector

    def word(n: int) -> str:
        """Return the n-th 32-byte word from calldata as hex (0-indexed)."""
        start = n * 64
        end = start + 64
        if len(calldata) >= end:
            return calldata[start:end]
        return ""

    def addr(w: str) -> str:
        """Extract Ethereum address from a padded 32-byte word."""
        return "0x" + w[-40:].lower() if w else ""

    def uint256(w: str) -> int:
        return int(w, 16) if w else 0

    try:
        if selector == "0xb6b55f25":  # deposit(uint256)
            fields["amount_wei"] = uint256(word(0))
            fields["amount_eth"] = fields["amount_wei"] / 1e18

        elif selector == "0x6dc4fc4e":  # resolveSingleBidAuction((address,uint256,bytes))
            # struct Bid: address expressLaneController, uint256 amount, bytes signature
            # ABI encoding: tuple offset (word0) then tuple contents (word1+)
            # with a dynamic bytes field, tuple is at offset pointed by word0
            # For a single struct arg, offset=0x20 typically; contents start at word1
            fields["expressLaneController"] = addr(word(1))
            fields["amount_wei"] = uint256(word(2))
            fields["amount_eth"] = fields["amount_wei"] / 1e18
            # signature offset at word3, length at word4, bytes at word5+
            sig_offset_words = uint256(word(3)) // 32
            sig_len = uint256(word(sig_offset_words + 1)) if word(sig_offset_words + 1) else 0
            sig_start = (sig_offset_words + 2) * 64
            fields["signature_len"] = sig_len
            if sig_len and len(calldata) >= sig_start + sig_len * 2:
                fields["signature_hex"] = "0x" + calldata[sig_start:sig_start + sig_len * 2]

        elif selector == "0x447a709e":  # resolveMultiBidAuction((Bid),(Bid))
            # Two struct args, each with dynamic bytes
            # First bid starts at dynamic offset word0=0x40, second at word1=offset
            # Simplified: grab the two controller addresses and amounts
            # bid1: tuple offset=0x40 → starts at word2
            fields["first_expressLaneController"] = addr(word(2))
            fields["first_amount_wei"] = uint256(word(3))
            fields["first_amount_eth"] = fields["first_amount_wei"] / 1e18
            # bid2 offset is word1 (e.g. 0x120), in words = 0x120//32 = 9
            second_offset_words = uint256(word(1)) // 32
            fields["second_expressLaneController"] = addr(word(second_offset_words))
            fields["second_amount_wei"] = uint256(word(second_offset_words + 1))
            fields["second_amount_eth"] = fields["second_amount_wei"] / 1e18

        elif selector == "0x007be2fe":  # transferExpressLaneController(uint64,address)
            fields["round"] = uint256(word(0))
            fields["newExpressLaneController"] = addr(word(1))

        elif selector == "0xbef0ec74":  # setTransferor((address,uint64))
            fields["transferor_addr"] = addr(word(1))
            fields["fixedUntilRound"] = uint256(word(2))

        elif selector == "0xb51d1d4f":  # initiateWithdrawal()
            pass  # no args

        elif selector == "0xc5b6aa2f":  # finalizeWithdrawal()
            pass  # no args

    except Exception as e:
        fields["_decode_error"] = str(e)

    return fields


def decode_auction_resolved_log(topics: List[str], data: str) -> Dict[str, Any]:
    """Decode AuctionResolved(bool,uint64,address,address,uint256,uint256,uint64,uint64)."""
    out: Dict[str, Any] = {}
    try:
        # topics[0] = event sig, topics[1] = isMultiBidAuction (indexed bool)
        # topics[2] = firstPriceBidder (indexed address)
        # topics[3] = firstPriceExpressLaneController (indexed address)
        if len(topics) >= 4:
            out["isMultiBidAuction"] = topics[1] != "0x" + "00" * 31 + "00"
            out["firstPriceBidder"] = "0x" + topics[2][-40:]
            out["firstPriceExpressLaneController"] = "0x" + topics[3][-40:]
        # data: round(uint64), firstPriceAmount, price, roundStart, roundEnd
        d = data.replace("0x", "")
        if len(d) >= 64 * 5:
            out["round"] = int(d[0:64], 16)
            out["firstPriceAmount_wei"] = int(d[64:128], 16)
            out["price_paid_wei"] = int(d[128:192], 16)
            out["roundStartTimestamp"] = int(d[192:256], 16)
            out["roundEndTimestamp"] = int(d[256:320], 16)
    except Exception as e:
        out["_decode_error"] = str(e)
    return out


def decode_set_elc_log(topics: List[str], data: str) -> Dict[str, Any]:
    """Decode SetExpressLaneController(uint64,address,address,address,uint64,uint64)."""
    out: Dict[str, Any] = {}
    try:
        if len(topics) >= 4:
            out["previousExpressLaneController"] = "0x" + topics[2][-40:]
            out["newExpressLaneController"] = "0x" + topics[3][-40:]
        d = data.replace("0x", "")
        if len(d) >= 64 * 3:
            out["round"] = int(d[0:64], 16)
            out["startTimestamp"] = int(d[64:128], 16)
            out["endTimestamp"] = int(d[128:192], 16)
    except Exception as e:
        out["_decode_error"] = str(e)
    return out


# ─────────────────────────────────────────────────────────────────────────────
# Output row builder
# ─────────────────────────────────────────────────────────────────────────────

def build_row(
    tx: Dict,
    block: Dict,
    match_source: str,
    logs: Optional[List[Dict]] = None,
    priority_flag: bool = False,
) -> Dict[str, Any]:
    selector, func_name = decode_selector(tx.get("input", ""))
    decoded = decode_high_level_fields(tx.get("input", ""), selector)

    # Decode relevant event logs if present
    decoded_logs: List[Dict] = []
    if logs:
        for log in logs:
            if not log.get("topics"):
                continue
            t0 = log["topics"][0].lower()
            event_name = EVENT_TOPICS.get(t0, "")
            if not event_name:
                continue
            log_row: Dict[str, Any] = {
                "logIndex": hex_to_int(log.get("logIndex")),
                "event": event_name,
                "topic0": t0,
            }
            if t0 == "0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47":
                log_row.update(decode_auction_resolved_log(log["topics"], log.get("data", "0x")))
            elif t0 == "0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b":
                log_row.update(decode_set_elc_log(log["topics"], log.get("data", "0x")))
            decoded_logs.append(log_row)

    row: Dict[str, Any] = {
        "txHash": tx.get("hash"),
        "block": hex_to_int(tx.get("blockNumber") or block.get("number")),
        "timestamp": hex_to_int(block.get("timestamp")),
        "from": tx.get("from", "").lower(),
        "to": (tx.get("to") or "").lower(),
        "value_wei": hex_to_int(tx.get("value")),
        "selector": selector,
        "function_name": func_name,
        "decoded_fields": decoded,
        "match_source": match_source,  # how this tx was identified
        "priority_flag": priority_flag,  # true = high-value auction action
        "logs": decoded_logs,
        "gas_used": None,  # populated from receipt if available
        "tx_index": hex_to_int(tx.get("transactionIndex")),
        # L2-specific (populated in l2-timeboosted mode)
        "timeboosted": tx.get("timeboosted", None),
    }
    return row


# ─────────────────────────────────────────────────────────────────────────────
# Mode: L1 Auction Contract scan
# ─────────────────────────────────────────────────────────────────────────────

def scan_l1_auction(
    rpc_url: str,
    contract_addr: str,
    from_block: int,
    to_block: int,
    out_file,
    batch_size: int = 50,
    rate_limit: float = 0.1,
) -> int:
    """
    Two-pass approach:
      1. eth_getLogs for all events from the contract in the range → gives txHashes quickly.
      2. For each unique block touched, fetch full block to get tx metadata.
      3. eth_getTransactionReceipt for logs (already from eth_getLogs, so skip).
    """
    contract_lower = contract_addr.lower()
    count = 0

    print(f"[l1-auction] Fetching logs from block {from_block} to {to_block} ...")

    # Step 1: event log scan
    log_results = rpc(rpc_url, "eth_getLogs", [{
        "fromBlock": hex(from_block),
        "toBlock": hex(to_block),
        "address": contract_addr,
    }])
    log_by_txhash: Dict[str, List[Dict]] = {}
    for log in (log_results or []):
        txh = log.get("transactionHash", "")
        log_by_txhash.setdefault(txh, []).append(log)

    # Step 2: direct tx scan (to catch call-only txs with no events, e.g. deposit)
    tx_hashes_from_logs: set = set(log_by_txhash.keys())

    # Collect all blocks touched
    blocks_touched = set()
    for log in (log_results or []):
        blocks_touched.add(hex_to_int(log.get("blockNumber")))

    # Scan each block for txs to the contract
    block_cache: Dict[int, Dict] = {}
    for bn in sorted(blocks_touched):
        blk = rpc(rpc_url, "eth_getBlockByNumber", [hex(bn), True])
        if not blk:
            continue
        block_cache[bn] = blk
        time.sleep(rate_limit)

    # Also scan blocks in range that might have had direct calls with no events
    # (e.g. pure view calls txs won't have events — but those are eth_calls not txs)
    # We only scan blocks where we already know events occurred (to avoid full block scan).
    # For a full scan (slower but complete), use --full-block-scan flag.

    # Combine: all txs from event-touched blocks that went TO the contract
    all_tx_to_contract: Dict[str, tuple] = {}  # txhash → (tx, block)
    for bn, blk in block_cache.items():
        for tx in blk.get("transactions", []):
            if (tx.get("to") or "").lower() == contract_lower:
                all_tx_to_contract[tx["hash"]] = (tx, blk)

    # Also add txs we found via log scan but aren't yet in all_tx_to_contract
    # (shouldn't happen, but defensive)
    for txh in tx_hashes_from_logs:
        if txh not in all_tx_to_contract:
            # Fetch individually
            tx = rpc(rpc_url, "eth_getTransactionByHash", [txh])
            if tx:
                bn = hex_to_int(tx.get("blockNumber"))
                if bn not in block_cache:
                    blk = rpc(rpc_url, "eth_getBlockByNumber", [hex(bn), False])
                    block_cache[bn] = blk or {}
                all_tx_to_contract[txh] = (tx, block_cache.get(bn, {}))
            time.sleep(rate_limit)

    print(f"[l1-auction] Found {len(all_tx_to_contract)} txs to contract, "
          f"{len(log_results or [])} event logs.")

    for txh, (tx, blk) in all_tx_to_contract.items():
        selector, _ = decode_selector(tx.get("input", ""))
        priority = selector in AUCTION_SELECTORS
        row = build_row(
            tx=tx,
            block=blk,
            match_source="l1_auction_call",
            logs=log_by_txhash.get(txh, []),
            priority_flag=priority,
        )
        out_file.write(json.dumps(row) + "\n")
        count += 1

    return count


# ─────────────────────────────────────────────────────────────────────────────
# Mode: L2 Timeboosted TX scan
# ─────────────────────────────────────────────────────────────────────────────

def scan_l2_timeboosted(
    rpc_url: str,
    from_block: int,
    to_block: int,
    out_file,
    rate_limit: float = 0.05,
) -> int:
    """
    Fetches each block and checks for the `timeboosted` flag on transactions.
    Arbitrum Nitro exposes this via eth_getBlockByNumber (with fullTxObjects=true).
    The `timeboosted` field is a boolean added to the tx object by Nitro when
    the tx was submitted through the express lane.

    Also checks for transactions TO the ExpressLaneAuction contract on L2
    (in case it's deployed there too, e.g. for cross-chain scenarios).
    """
    count = 0
    print(f"[l2-timeboosted] Scanning blocks {from_block} to {to_block} ...")

    for bn in range(from_block, to_block + 1):
        try:
            blk = rpc(rpc_url, "eth_getBlockByNumber", [hex(bn), True])
        except Exception as e:
            print(f"  [warn] block {bn}: {e}", file=sys.stderr)
            time.sleep(1)
            continue

        if not blk:
            continue

        for tx in blk.get("transactions", []):
            is_timeboosted = tx.get("timeboosted", False)
            to_addr = (tx.get("to") or "").lower()
            is_auction_call = to_addr == EXPR_LANE_AUCTION_MAINNET.lower()

            if not is_timeboosted and not is_auction_call:
                continue

            source = []
            if is_timeboosted:
                source.append("l2_timeboosted_flag")
            if is_auction_call:
                source.append("l2_auction_call")

            selector, _ = decode_selector(tx.get("input", ""))
            priority = selector in AUCTION_SELECTORS or is_timeboosted

            row = build_row(
                tx=tx,
                block=blk,
                match_source="|".join(source),
                priority_flag=priority,
            )
            out_file.write(json.dumps(row) + "\n")
            count += 1

        if bn % 100 == 0:
            print(f"  ... block {bn}, found {count} so far")
        time.sleep(rate_limit)

    return count


# ─────────────────────────────────────────────────────────────────────────────
# Mode: dataset (pre-fetched JSON)
# ─────────────────────────────────────────────────────────────────────────────

def scan_dataset(dataset_path: str, contract_addr: str, out_file) -> int:
    """
    Reads a JSON file containing either:
      - A list of transaction objects, or
      - A list of block objects (each with .transactions[])
    Filters for express-lane candidates.
    """
    contract_lower = contract_addr.lower()
    count = 0

    with open(dataset_path) as f:
        data = json.load(f)

    # Normalize to list of txs
    txs: List[Dict] = []
    if isinstance(data, list):
        for item in data:
            if "transactions" in item:
                # block object
                for tx in item["transactions"]:
                    tx["_block"] = item
                    txs.append(tx)
            elif "hash" in item:
                txs.append(item)

    print(f"[dataset] Processing {len(txs)} transactions from {dataset_path} ...")

    for tx in txs:
        to_addr = (tx.get("to") or "").lower()
        is_auction_call = to_addr == contract_lower
        is_timeboosted = tx.get("timeboosted", False)

        if not is_auction_call and not is_timeboosted:
            continue

        blk = tx.pop("_block", {})
        source = []
        if is_auction_call:
            source.append("dataset_auction_call")
        if is_timeboosted:
            source.append("dataset_timeboosted")

        selector, _ = decode_selector(tx.get("input", ""))
        row = build_row(
            tx=tx,
            block=blk,
            match_source="|".join(source),
            priority_flag=selector in AUCTION_SELECTORS or is_timeboosted,
        )
        out_file.write(json.dumps(row) + "\n")
        count += 1

    return count


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Arbitrum Timeboost/ExpressLane TX filter pipeline",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("--mode", choices=["l1-auction", "l2-timeboosted", "dataset"],
                        required=True, help="Ingestion mode")
    parser.add_argument("--rpc", default=os.environ.get("ETH_RPC", ""),
                        help="JSON-RPC endpoint URL")
    parser.add_argument("--contract",
                        default=EXPR_LANE_AUCTION_MAINNET,
                        help=f"ExpressLaneAuction contract address (default: {EXPR_LANE_AUCTION_MAINNET})")
    parser.add_argument("--from-block", type=int, default=0,
                        help="Start block (inclusive)")
    parser.add_argument("--to-block", type=int, default=0,
                        help="End block (inclusive)")
    parser.add_argument("--dataset", default="",
                        help="Path to pre-fetched JSON dataset (for --mode dataset)")
    parser.add_argument("--out", default="expresslane_candidates.jsonl",
                        help="Output JSONL file path")
    parser.add_argument("--rate-limit", type=float, default=0.1,
                        help="Seconds to sleep between RPC calls (default: 0.1)")
    parser.add_argument("--batch-size", type=int, default=50,
                        help="Log fetch batch size (default: 50)")
    parser.add_argument("--append", action="store_true",
                        help="Append to output file instead of overwriting")
    args = parser.parse_args()

    # Validate
    if args.mode in ("l1-auction", "l2-timeboosted") and not args.rpc:
        parser.error("--rpc is required for modes l1-auction and l2-timeboosted")
    if args.mode == "dataset" and not args.dataset:
        parser.error("--dataset is required for mode dataset")
    if args.mode in ("l1-auction", "l2-timeboosted"):
        if args.from_block == 0 and args.to_block == 0:
            parser.error("--from-block and --to-block are required")

    open_mode = "a" if args.append else "w"
    with open(args.out, open_mode) as out_file:
        if args.mode == "l1-auction":
            count = scan_l1_auction(
                rpc_url=args.rpc,
                contract_addr=args.contract,
                from_block=args.from_block,
                to_block=args.to_block,
                out_file=out_file,
                batch_size=args.batch_size,
                rate_limit=args.rate_limit,
            )
        elif args.mode == "l2-timeboosted":
            count = scan_l2_timeboosted(
                rpc_url=args.rpc,
                from_block=args.from_block,
                to_block=args.to_block,
                out_file=out_file,
                rate_limit=args.rate_limit,
            )
        elif args.mode == "dataset":
            count = scan_dataset(
                dataset_path=args.dataset,
                contract_addr=args.contract,
                out_file=out_file,
            )
        else:
            parser.error(f"Unknown mode: {args.mode}")

    print(f"\n✓ Done. {count} candidate transaction(s) written to {args.out}")


if __name__ == "__main__":
    main()
