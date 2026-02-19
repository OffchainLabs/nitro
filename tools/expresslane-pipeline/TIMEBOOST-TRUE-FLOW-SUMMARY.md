# Timeboost True Flow Summary (2026-02-19)

## ✅ Deliverables
- **New scripts**:
  - `timeboost_receipt_extractor.py` — receipt-based timeboosted extractor (supports metadata prefilter + auto-detect auction windows)
  - `timeboost_scan_last_auctions.py` — practical scanner for the last N auction rounds
- **True timeboosted corpus (JSONL)**:
  - `output/timeboosted_true_20260219_055159_last10.jsonl` (381 rows)
- **Stats JSON**:
  - `output/timeboosted_true_20260219_055159_last10_stats.json`

## Key Results (last 10 auction rounds)
- **Total true timeboosted txs:** 381
- **Blocks scanned:** 2,600 (10 rounds × 260 blocks)
- **Blocks with timeboosted txs:** 10+

### Top Selectors
```
0xb2460c48  232
0x6bfd6286   69
0xb42ebb63   33
0x000000e7   20
0x00000000   11
```

### Top Contracts (destination)
```
0x27920e8039d2b6e93e36f5d5f53b998e2e631a70  343
0x96daa0b8a5499ea9323421ed0cda06b345caab73   20
0xee2e7bbb67676292af2e31dffd1fea2276d6c7ba   10
```

### Top Senders
```
0x9e6441c6d930f2e98bfb6cae6dc46729c862055f  20
0x13bc345dbfa3e0d165cff8741d55f285daf81698   6
0x954cbe6534d6fce17c89c278d020d131a34cf475   6
```

## Data Quality Caveats
- **arb_getRawBlockMetadata returned all-zero metadata** across the 58k-block auction-active range (433,444,240–433,503,000).
  - This suggests the public RPC either strips the boosted bitmask or the metadata format differs.
  - As a result, **receipt-based detection is currently the only reliable method** on public RPC.
- **Timeboosted flag *is* present in `eth_getBlockReceipts`** on Nitro public RPC; this is the authoritative signal.
- Only the **last 10 auction rounds** were scanned here for practicality (2,600 blocks). This is a **meaningful recent sample**, but not full history.

## Exact Commands (Repro)
```bash
# 1) Collect AuctionResolved blocks (cached in /tmp/auction_blocks.json)
python3 - <<'PY'
import requests, json, time
RPC = 'https://arb1.arbitrum.io/rpc'
AUC_TOPIC = '0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47'
AUCTION = '0x5fcb496A31b7ae91E7c9078EC662bD7a55cD3079'

s = requests.Session(); s.headers.update({'Content-Type':'application/json','User-Agent':'Mozilla/5.0'})
_id=0
def rpc(m,p,t=20):
  global _id; _id+=1
  r=s.post(RPC,json={'jsonrpc':'2.0','id':_id,'method':m,'params':p},timeout=t)
  r.raise_for_status(); d=r.json(); return d.get('result')

latest = int(rpc('eth_blockNumber', []), 16)
all_logs = []
for chunk_start in range(latest-200000, latest+1, 50000):
  chunk_end = min(chunk_start + 49999, latest)
  logs = rpc('eth_getLogs', [{'fromBlock':hex(chunk_start),'toBlock':hex(chunk_end),
                              'address':AUCTION,'topics':[AUC_TOPIC]}], 30)
  if logs: all_logs.extend(logs)
  time.sleep(0.2)

auction_blocks = [int(l['blockNumber'],16) for l in all_logs]
json.dump({'latest':latest,'auction_blocks':auction_blocks,'count':len(auction_blocks)},
          open('/tmp/auction_blocks.json','w'))
PY

# 2) Scan last 10 auction rounds for true timeboosted txs (receipt-based)
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
python3 timeboost_scan_last_auctions.py \
  --auction-file /tmp/auction_blocks.json \
  --last-n 10 \
  --window 260 \
  --rate 0.05 \
  --out output/timeboosted_true_${TIMESTAMP}_last10.jsonl
```

## Status
**READY_WITH_TRUE_FLOW_DATA**
