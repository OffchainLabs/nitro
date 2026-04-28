#!/usr/bin/env python3
"""Build sepolia_block_101_809_176.json regression fixture.

Pulls live state from Arbitrum Sepolia archive RPC and emits a self-contained
ExecutionFixture that reproduces the divergent log[7] data word reported for
tx 0x6cbe9345...
"""
import base64
import json
import os
import subprocess
import sys
from pathlib import Path

from Crypto.Hash import keccak

REPO = Path(__file__).resolve().parents[5]
ENV_FILE = REPO / ".env.alchemy"
TX_HASH = "0x6cbe9345bc144ee89c0200ea8de5a3b8c3776deaf88e63fc17d17edae996f3b4"
TARGET_BLOCK = 0x6117C18  # 101_809_176
PARENT_BLOCK = 0x6117C17
ARBOS_STATE_ADDR = "0xa4b05fffffffffffffffffffffffffffffffffff"
SEQUENCER_ADDR = "0xa4b000000000000000000073657175656e636572"
DEPOSIT_REQUEST_ID = "0x" + "00" * 31 + "01"

OUT_PATH = Path(__file__).resolve().parent / "sepolia_block_101_809_176.json"


def kec(data):
    h = keccak.new(digest_bits=256)
    h.update(data)
    return h.digest()


def derive_sub_key(parent: bytes, sub: bytes) -> bytes:
    """Mirror arb-storage's derive_sub_key (B256::ZERO base treated as empty)."""
    if parent == bytes(32):
        return kec(sub)
    return kec(parent + sub)


def storage_key_map(base_key: bytes, offset: int) -> bytes:
    key_bytes = bytearray(32)
    key_bytes[24:32] = offset.to_bytes(8, "big")
    h = kec(base_key + bytes(key_bytes[:31]))
    return h[:31] + bytes([key_bytes[31]])


def storage_key_map_b256(base_key: bytes, key: bytes) -> bytes:
    h = kec(base_key + key[:31])
    return h[:31] + bytes([key[31]])


def alchemy_url() -> str:
    if not ENV_FILE.exists():
        raise SystemExit(f"missing {ENV_FILE}")
    for line in ENV_FILE.read_text().splitlines():
        if line.startswith("ARB_SEPOLIA_RPC="):
            return line.split("=", 1)[1].strip()
    raise SystemExit("ARB_SEPOLIA_RPC not in .env.alchemy")


def rpc(url: str, method: str, params):
    body = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
    out = subprocess.check_output(
        [
            "curl",
            "-sS",
            "-X",
            "POST",
            "-H",
            "Content-Type: application/json",
            "--data",
            body,
            url,
        ],
        timeout=120,
    )
    d = json.loads(out)
    if "error" in d:
        raise RuntimeError(f"{method}: {d['error']}")
    return d["result"]


def hex_to_bytes(s: str) -> bytes:
    return bytes.fromhex(s[2:] if s.startswith("0x") else s)


def main():
    url = alchemy_url()
    print(f"-> RPC: {url}", file=sys.stderr)

    print("-> fetching tx + receipt", file=sys.stderr)
    raw_tx_hex = rpc(url, "eth_getRawTransactionByHash", [TX_HASH])
    receipt = rpc(url, "eth_getTransactionReceipt", [TX_HASH])
    tx = rpc(url, "eth_getTransactionByHash", [TX_HASH])
    target_block = rpc(url, "eth_getBlockByNumber", [hex(TARGET_BLOCK), False])

    sender = tx["from"].lower()
    print(f"   sender={sender} nonce=0x{int(tx['nonce'], 16):x}", file=sys.stderr)

    # The alchemy block returns extra Arbitrum fields.
    l1_block_number = int(target_block.get("l1BlockNumber", "0x0"), 16)
    l2_timestamp = int(target_block["timestamp"], 16)

    print("-> debug_traceTransaction(prestateTracer)", file=sys.stderr)
    prestate = rpc(url, "debug_traceTransaction", [TX_HASH, {"tracer": "prestateTracer"}])

    # Pull contract code @ parent for everything in the prestate that has code.
    print("-> fetching code @ parent for all prestate addrs", file=sys.stderr)
    code_for = {}
    for addr, info in prestate.items():
        if info.get("code"):
            code_for[addr] = rpc(url, "eth_getCode", [addr, hex(PARENT_BLOCK)])
        else:
            code_for[addr] = "0x"

    # Build alloc:
    alloc = {}
    sender_balance = int(prestate[sender]["balance"], 16) if sender in prestate else 10**21
    # Top up sender so the gas reserve matches reality.
    alloc[sender] = {
        "balance": hex(sender_balance),
        "nonce": "0x0",
    }

    # Each contract: code only (storage handled below for the proxy).
    for addr, info in prestate.items():
        if addr == sender:
            continue
        if addr == ARBOS_STATE_ADDR:
            continue
        entry = {
            "balance": info.get("balance", "0x0"),
        }
        nonce = info.get("nonce")
        if nonce is not None and nonce != 0:
            entry["nonce"] = hex(nonce)
        if info.get("code"):
            entry["code"] = info["code"]
        if info.get("storage"):
            # Convert storage hex strings to {slot: value} dict.
            entry["storage"] = info["storage"]
        alloc[addr] = entry

    # ── ArbOS state: fold the prestate slots in at 0xa4b05fff... ──
    arbos_state = prestate.get(ARBOS_STATE_ADDR, {})
    arbos_storage = dict(arbos_state.get("storage", {}))

    # Build the alloc entry for the ArbOS state account. The chainspec parser
    # will merge our user-supplied storage with the bootstrap-injected values.
    alloc[ARBOS_STATE_ADDR] = {
        "balance": arbos_state.get("balance", "0x0"),
        "nonce": hex(arbos_state.get("nonce", 1) or 1),
        "storage": arbos_storage,
    }

    # ── Build the L1 messages ──
    fund_amount = max(2 * 10**18, sender_balance + 10**18)
    deposit = {
        "msgIdx": None,
        "message": {
            "header": {
                "kind": 12,  # ETH deposit
                "sender": sender,
                "blockNumber": l1_block_number,
                "timestamp": l2_timestamp,
                "requestId": DEPOSIT_REQUEST_ID,
                "baseFeeL1": 0,
            },
            "l2Msg": base64.b64encode(
                hex_to_bytes(sender) + fund_amount.to_bytes(32, "big")
            ).decode(),
        },
        "delayedMessagesRead": 1,
    }

    raw_tx = hex_to_bytes(raw_tx_hex)
    body = bytes([0x04]) + raw_tx
    user_tx_msg = {
        "msgIdx": None,
        "message": {
            "header": {
                "kind": 3,  # L2 message
                "sender": SEQUENCER_ADDR,
                "blockNumber": l1_block_number,
                "timestamp": l2_timestamp,
                "baseFeeL1": 0,
            },
            "l2Msg": base64.b64encode(body).decode(),
        },
        "delayedMessagesRead": 1,
    }

    # ── Expected logs (canonical) ──
    expected_logs = []
    for i, log in enumerate(receipt["logs"]):
        expected_logs.append(
            {
                "address": log["address"].lower(),
                "topics": log["topics"],
                "data": log["data"],
                "txHash": log["transactionHash"],
                "logIndex": int(log["logIndex"], 16),
            }
        )

    # The fixture's first L2 block corresponds to message[0] (deposit).
    # Block 1 = funded EOA. Block 2 = the user tx.
    fixture = {
        "name": "sepolia_block_101_809_176",
        "description": (
            "Regression: Stylus contract at Arbitrum Sepolia block 101,809,176 "
            "produces wrong log[7] data word vs canonical Nitro. Tx 0x6cbe9345..."
            "Selector 0x70aeb617(uint256,uint256,uint256). 6 Stylus contracts pre-"
            "alloc'd; ArbOS program metadata + module hashes + l1/l2 pricing slots "
            "loaded from prestateTracer."
        ),
        "genesis": {
            "config": {
                "chainId": 421614,
                "homesteadBlock": 0,
                "daoForkSupport": True,
                "eip150Block": 0,
                "eip155Block": 0,
                "eip158Block": 0,
                "byzantiumBlock": 0,
                "constantinopleBlock": 0,
                "petersburgBlock": 0,
                "istanbulBlock": 0,
                "muirGlacierBlock": 0,
                "berlinBlock": 0,
                "londonBlock": 0,
                "depositContractAddress": "0x0000000000000000000000000000000000000000",
                "clique": {"period": 0, "epoch": 0},
                "arbitrum": {
                    "EnableArbOS": True,
                    "AllowDebugPrecompiles": True,
                    "DataAvailabilityCommittee": False,
                    "InitialArbOSVersion": 32,
                    "InitialChainOwner": "0x0000000000000000000000000000000000000000",
                    "GenesisBlockNum": 0,
                },
            },
            "alloc": alloc,
            "coinbase": "0x0000000000000000000000000000000000000000",
            "difficulty": "0x1",
            "extraData": "0x" + "00" * 32,
            "gasLimit": "0x4000000000000",
            "nonce": "0x1",
            "timestamp": "0x0",
        },
        "messages": [deposit, user_tx_msg],
        "expected": {
            "blocks": [],
            "eth_calls": [],
            "storage": [],
            "balances": [],
            "txReceipts": [
                {
                    "txHash": TX_HASH,
                    "blockNumber": 2,
                    "status": int(receipt["status"], 16),
                    "gasUsed": int(receipt["gasUsed"], 16),
                    "from": receipt["from"].lower(),
                    "to": receipt["to"].lower(),
                    "logs": expected_logs,
                }
            ],
        },
    }

    # Pretty print.
    OUT_PATH.write_text(json.dumps(fixture, indent=2) + "\n")
    print(f"-> wrote {OUT_PATH} ({OUT_PATH.stat().st_size} bytes)", file=sys.stderr)


if __name__ == "__main__":
    main()
