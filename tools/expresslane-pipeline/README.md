# Arbitrum Timeboost / Express Lane TX Filter Pipeline

**Status: READY_FOR_COLLECTION**  
**Author:** TB-3 subagent — expresslane-filter-pipeline  
**Date:** 2026-02-19  
**Workspace:** `/mnt/volume-hel1-1/bug-hunting/arbitrum-nitro/tools/expresslane-pipeline/`

---

## Purpose

First-step data pipeline that identifies and extracts all candidate
Timeboost/Express Lane related transactions from on-chain data.
No abuse/mismatch analysis is done here — this layer is purely identification
and structured extraction, ready to feed into downstream analysis scripts.

---

## Architecture Overview

```
                ┌─────────────────────────────────────────────┐
                │             expresslane_filter.py           │
                │                                             │
 JSON-RPC ─────►│  [Mode: l1-auction]  eth_getLogs +          │
                │   eth_getBlockByNumber  for contract txs    │──► JSONL rows
 JSON-RPC ─────►│  [Mode: l2-timeboosted] scan tx.timeboosted │
                │   flag on each Arbitrum block               │
 Dataset ──────►│  [Mode: dataset] ingest pre-fetched JSON    │
                └─────────────────────────────────────────────┘
```

**Two detection planes:**

| Plane | What it catches | Signal strength |
|-------|----------------|-----------------|
| **L1-Auction (primary)** | All calls/events on `ExpressLaneAuction` contract (Arbitrum One L2) | **Definitive** — auction lifecycle, bid resolution, controller assignment |
| **L2-Timeboosted (secondary)** | Transactions with `tx.timeboosted == true` on Arbitrum One blocks | **Definitive** — txs actually sequenced through the express lane |

---

## Contract Details (Verified)

| Field | Value |
|-------|-------|
| Contract | `ExpressLaneAuction` (TransparentUpgradeableProxy) |
| Address (Arbitrum One) | `0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079` |
| Bidding token | WETH — `0x980B62Da83eFf3D4576C647993b0c1D7faf17c73` |
| Auctioneer (deployer EOA) | `0xeee584DA928A94950E177235EcB9A99bb655c7A0` |
| Round duration | 60 seconds |
| Auction close window | 15 seconds before round end |
| Domain separator | `0x1d0408ba738f79bf952118f76d58c76ac0b551f17e6e846570a7766a2865ae7a` |

> Observed auctioneer on-chain (block range 420M): `0x28452b38064b1dc5e5e2ae4c1be5d4c392f38dcf`  
> (differs from deployer — the auctioneer role can be updated via access control)

---

## Function Selectors (4-byte)

| Selector | Function | Category |
|----------|----------|----------|
| `0x6dc4fc4e` | `resolveSingleBidAuction((address,uint256,bytes))` | ⭐ Auction resolution |
| `0x447a709e` | `resolveMultiBidAuction((address,uint256,bytes),(address,uint256,bytes))` | ⭐ Auction resolution |
| `0x007be2fe` | `transferExpressLaneController(uint64,address)` | ⭐ Controller transfer |
| `0xb6b55f25` | `deposit(uint256)` | Lifecycle |
| `0xb51d1d4f` | `initiateWithdrawal()` | Lifecycle |
| `0xc5b6aa2f` | `finalizeWithdrawal()` | Lifecycle |
| `0x6ad72517` | `flushBeneficiaryBalance()` | Admin |
| `0xbef0ec74` | `setTransferor((address,uint64))` | Controller config |
| `0x0d253fbe` | `resolvedRounds()` | View |
| `0xf698da25` | `domainSeparator()` | View |
| `0xce9c7c0d` | `setReservePrice(uint256)` | Admin |
| `0xe4d20c1d` | `setMinReservePrice(uint256)` | Admin |
| `0x1c31f710` | `setBeneficiary(address)` | Admin |
| `0xfed87be8` | `setRoundTimingInfo((int64,uint64,uint64,uint64))` | Admin |

---

## Event Topics (topic0)

| Topic0 | Event | Notes |
|--------|-------|-------|
| `0x7f5bdab...` | `AuctionResolved(bool,uint64,address,address,uint256,uint256,uint64,uint64)` | ⭐ Every round resolution |
| `0xb59adc8...` | `SetExpressLaneController(uint64,address,address,address,uint64,uint64)` | ⭐ Controller assignment |
| `0xe1fffcc...` | `Deposit(address,uint256)` | Bidder deposits |
| `0x31f6920...` | `WithdrawalInitiated(address,uint256,uint256)` | TB-1 TOCTOU vector |
| `0x9e5c4f9...` | `WithdrawalFinalized(address,uint256)` | Funds exit |
| `0xf6d28df...` | `SetTransferor(address,address,uint64)` | Controller delegation |
| `0x9725e37...` | `SetReservePrice(uint256,uint256)` | Admin action |
| `0x5848068...` | `SetMinReservePrice(uint256,uint256)` | Admin action |
| `0x8a0149b...` | `SetBeneficiary(address,address)` | Admin action |

---

## Output Schema

Each JSONL row contains:

```json
{
  "txHash":          "0x...",          // L1/L2 transaction hash
  "block":           420000118,        // block number (int)
  "timestamp":       1768076438,       // unix timestamp (int)
  "from":            "0x...",          // caller address (lowercase)
  "to":              "0x...",          // contract address (lowercase)
  "value_wei":       0,                // ETH value sent (usually 0 — bidding token is ERC20)
  "selector":        "0x6dc4fc4e",     // 4-byte function selector
  "function_name":   "resolveSingleBidAuction(...)",
  "decoded_fields":  {                 // best-effort ABI decode
    "expressLaneController": "0x...",
    "amount_wei":     1357525220868480,
    "amount_eth":     0.001357,
    "signature_len":  65,
    "signature_hex":  "0x..."
  },
  "match_source":    "l1_auction_call",  // how this tx was found
  "priority_flag":   true,              // true = auction resolution or controller action
  "logs": [                            // decoded event logs for this tx
    {
      "logIndex":    0,
      "event":       "SetExpressLaneController(...)",
      "topic0":      "0xb59a...",
      "round":       439823,
      "newExpressLaneController": "0x...",
      ...
    }
  ],
  "gas_used":        null,              // populated if receipt was fetched
  "tx_index":        1,                 // position in block
  "timeboosted":     null               // bool if l2-timeboosted mode, else null
}
```

---

## Installation

```bash
# Python 3.8+ required
pip3 install requests  # or use stdlib urllib fallback (no pip needed)
```

No additional dependencies. The script uses only `requests` (with urllib fallback).

---

## Usage

### Mode 1: L1-Auction (recommended primary source)

Scans all transactions **to the ExpressLaneAuction contract** by:
1. Fetching all event logs from the contract in the block range
2. Resolving unique blocks touched
3. Collecting all txs in those blocks that target the contract
4. Decoding selectors and event logs

```bash
python3 expresslane_filter.py \
  --mode l1-auction \
  --rpc https://arb1.arbitrum.io/rpc \
  --contract 0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079 \
  --from-block 420000000 \
  --to-block 420010000 \
  --out output.jsonl \
  --rate-limit 0.2
```

**Required inputs:**
- `--rpc` — Arbitrum One JSON-RPC endpoint (public: `https://arb1.arbitrum.io/rpc`)
- `--from-block` / `--to-block` — block range to scan
- `--contract` — default is the mainnet address above

**Tip — finding active block ranges:**  
Timeboost went live in late 2024. Block `~370M` (Sep 2024) is a safe start. Current
block is ~433M. One round = ~240 Arbitrum blocks (60s × 4 blocks/s).

---

### Mode 2: L2-Timeboosted

Fetches every block in range and checks for `tx.timeboosted == true`. This is
the **only way** to catch express-lane user transactions (not just auctioneer calls).

```bash
python3 expresslane_filter.py \
  --mode l2-timeboosted \
  --rpc https://arb1.arbitrum.io/rpc \
  --from-block 420000000 \
  --to-block 420001000 \
  --out timeboosted.jsonl \
  --rate-limit 0.05
```

> **Note:** Fetches full block with transactions for every block in range.  
> Rate limit accordingly — 1000 blocks = ~50 sec at default 0.05s rate.  
> For large ranges, use a private RPC endpoint to avoid throttling.

---

### Mode 3: Dataset (pre-fetched JSON)

Reads a local JSON file containing transaction or block objects.

```bash
python3 expresslane_filter.py \
  --mode dataset \
  --dataset /path/to/blocks.json \
  --contract 0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079 \
  --out filtered.jsonl
```

**Expected format:**
```json
[
  { "hash": "0x...", "to": "0x...", "input": "0x...", "blockNumber": "0x...", ... },
  { "number": "0x...", "timestamp": "0x...", "transactions": [...] }
]
```

---

## Sample Run

```
[l1-auction] Fetching logs from block 420000000 to 420010000 ...
[l1-auction] Found 42 txs to contract, 84 event logs.

✓ Done. 42 candidate transaction(s) written to sample_output.jsonl
```

See `sample_output.jsonl` for 42 real transactions from Arbitrum One block range 420M–420.01M.

**Stats from sample:**

| Function | Count |
|----------|-------|
| `resolveSingleBidAuction` | 36 |
| `resolveMultiBidAuction` | 6 |

- All 42 txs from a single auctioneer: `0x28452b38...`
- Bid amounts ranged: ~0.001–0.002 WETH per round
- Every round resolution emits `AuctionResolved` + `SetExpressLaneController`

---

## Filter Logic & Signal Strength

### Primary filters (zero false negatives for contract-level actions)

1. **`to == ExpressLaneAuction`** — catches all direct contract interactions
2. **Event log topic0 match** — catches all on-chain state changes

These two combined give a **complete** view of all on-chain Timeboost actions.
No express-lane lifecycle event can occur without touching the contract.

### Secondary filter (for user-submitted express-lane txs)

3. **`tx.timeboosted == true`** — set by the Nitro sequencer on any tx that was
   submitted via `timeboost_sendExpressLaneTransaction` RPC. This flag appears in
   Arbitrum One block data returned by standard `eth_getBlockByNumber`.

### Priority flags

`priority_flag = true` when:
- `selector in {0x6dc4fc4e, 0x447a709e, 0x007be2fe}` (auction resolution / controller transfer)
- OR `tx.timeboosted == true`

Non-priority (but still relevant): `deposit`, `initiateWithdrawal`, `finalizeWithdrawal`

---

## Precision Notes: False Positive / False Negative Risks

### False Positives

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Another contract has same address collision | Near-zero | Address is proxy with verified bytecode |
| `timeboosted` flag set incorrectly by sequencer bug | Very low | Cross-check against `AuctionResolved` round timestamps |
| Selector collision (another contract uses 0x6dc4fc4e) | Low for auction contract filter (we filter by `to` address) | N/A |
| Reverted txs included (failed resolutions) | Possible | Add receipt fetch + filter `status == 0x1` |

**Current known FP source:** The pipeline includes **reverted transactions**.  
A failed `resolveMultiBidAuction` (e.g., insufficient balance) will appear.  
To remove: add `--receipts` flag (not yet implemented) to fetch and filter by `status`.

### False Negatives

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Express-lane user txs missed in l1-auction mode | **Certain** — user txs don't call the auction contract | Use l2-timeboosted mode in parallel |
| Blocks missed if RPC returns error mid-scan | Low | Script retries 3× per call |
| Event logs beyond RPC range limit (some providers limit to 2000) | Medium for large ranges | Chunk into 1000-block segments |
| `timeboosted` flag absent on old Nitro versions | Low | Old txs pre-Timeboost activation won't have flag |
| Proxy contract interactions via delegatecall | Not applicable here | ExpressLaneAuction uses transparent proxy; logic is in implementation |

### Tuning Recommendations

- **For maximum coverage**: run both modes (`l1-auction` + `l2-timeboosted`) and merge on `txHash`
- **For speed**: `l1-auction` only covers the auctioneer transactions (42 per ~10k blocks)
- **For user activity**: `l2-timeboosted` is the only signal for controller-submitted txs
- **Block range chunks**: keep to ≤10,000 blocks per call for public RPCs
- **Rate limiting**: 0.1–0.2s works for public Arbitrum RPC; 0.01s for private/local RPC

---

## Next-Step Hooks for Deeper Analysis

The JSONL output is designed to feed directly into downstream scripts:

### 1. Abuse / Mismatch Detection (TB-3 focus)

```python
# Load pipeline output
rows = [json.loads(l) for l in open('output.jsonl')]

# Find auction rounds
auction_txs = [r for r in rows if r['selector'] in ('0x6dc4fc4e', '0x447a709e')]

# For each resolved round: extract controller, round#, timestamps
for tx in auction_txs:
    for log in tx['logs']:
        if 'AuctionResolved' in log.get('event',''):
            round_num   = log['round']
            controller  = log['firstPriceExpressLaneController']
            round_start = log['roundStartTimestamp']
            round_end   = log['roundEndTimestamp']
            price_paid  = log['price_paid_wei']
            # → now correlate with WithdrawalInitiated events to check TOCTOU
            # → compare price_paid vs firstPriceAmount for second-price logic
```

### 2. TOCTOU Balance Check (Finding 1 from TIMEBOOST_AUDIT.md)

```python
# Find WithdrawalInitiated events
withdrawals = []
for r in rows:
    for log in r['logs']:
        if 'WithdrawalInitiated' in log.get('event',''):
            withdrawals.append({
                'from': r['from'],
                'block': r['block'],
                'timestamp': r['timestamp'],
                'log': log
            })

# Cross-reference: was there an auction resolution shortly after a withdrawal?
# → Sort by timestamp and look for withdrawal → resolution sequences
```

### 3. Bid Replacement Attack (Finding 2)

Only visible via the off-chain bid validator Redis stream — not on-chain.
Hook: compare `firstPriceAmount` vs `price_paid` gap per round.
Large gaps = competitive auctions (not an attack signal by itself).

### 4. Controller Activity Analysis

```python
# Track which addresses win rounds
controllers = Counter()
for tx in auction_txs:
    for log in tx['logs']:
        if 'AuctionResolved' in log.get('event',''):
            controllers[log['firstPriceExpressLaneController']] += 1
# → Identify dominant controllers
# → Look for same bidder + controller → check if they're front-running
```

### 5. L2 Timeboosted TX Analysis

After running `l2-timeboosted` mode:
```python
timeboosted = [r for r in rows if r.get('timeboosted')]
# → Check tx.tx_index distribution (should be near top of block)
# → Cross-check sender matches the round's controller
# → Look for DontCareSequence abuse patterns (rapid fire txs from controller)
```

---

## Codebase Reference

| File | Purpose |
|------|---------|
| `timeboost/types.go` | `ExpressLaneSubmission` struct, signing scheme |
| `timeboost/bid_validator.go` | Off-chain bid validation + EIP-712 |
| `timeboost/auctioneer.go` | On-chain resolution calls |
| `timeboost/bid_cache.go` | In-memory bid store (keyed by controller) |
| `execution/gethexec/express_lane_service.go` | Sequencer-side express lane processing |
| `execution/gethexec/express_lane_tracker.go` | On-chain event polling for controller assignment |
| `solgen/go/express_lane_auctiongen/` | Go ABI bindings |
| `contracts/src/express-lane-auction/IExpressLaneAuction.sol` | Full interface + event definitions |
| `knowledge-v2/08-TIMEBOOST.md` | Detailed code walkthrough with security notes |
| `TIMEBOOST_AUDIT.md` | 14-finding audit report |
| `threat-model-v2-timeboost.md` | Cross-subsystem threat model |

---

## Deliverables

| Artifact | Path | Status |
|----------|------|--------|
| Filter script | `tools/expresslane-pipeline/expresslane_filter.py` | ✅ |
| README/docs | `tools/expresslane-pipeline/README.md` | ✅ |
| Sample output (42 real txs) | `tools/expresslane-pipeline/sample_output.jsonl` | ✅ |
| Selector/topic reference | Embedded in script + this README | ✅ |

---

**STATUS: READY_FOR_COLLECTION**
