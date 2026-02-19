# Selector Conditional Buckets (Timeboosted True Corpus)

**Dataset:** `output/timeboosted_true_20260219_055159_last10.jsonl` (381 rows)  
**Top selectors analyzed:** `0xb2460c48`, `0x6bfd6286`, `0xb42ebb63`, `0x000000e7`

## Executive Summary
All top selectors resolve to **custom swap routers/executors**. Three selectors (b2460c48, 6bfd6286, b42ebb63) target the **same contract** `0x27920e8039d2b6e93e36f5d5f53b998e2e631a70` and use **packed, non-ABI calldata** to drive **Uniswap V3 swaps** across a small set of pools (WETH/USDC, WETH/USDT, WBTC/USDT, WBTC/USDC, WETH/ARB, WETH/WBTC). Many txs revert (log_count=0), implying **conditional execution** (likely price/limit thresholds). The selector `0x000000e7` targets a separate contract `0x96daa0b8a5499ea9323421ed0cda06b345caab73` and uses a 5-word ABI-like payload referencing a **single UniV3 pool** (ETH/USDC) plus direction and limit parameters.

---

## Selector: `0xb2460c48` (count 232)
**Destination contract:** `0x27920e8039d2b6e93e36f5d5f53b998e2e631a70`  
**Input length:** 36 bytes total (4 selector + 32 packed data)  
**Log count distribution:** 4 logs (114), 0 logs (114), 3 logs (4)  
**Success pattern:** 4-log txs emit ERC20 Transfers + UniV3 Swap event; 0-log txs are **reverts**.

### Calldata structure (packed, non-ABI)
- 32-byte payload (after selector) with **market ID at byte[1]**.
- Constant byte[0]=0x00, byte[31]=0x01 in most samples.
- Bytes in the middle encode **amount/limit/price** in a compact format; the exact field layout is not fully recovered.

### Market ID mapping (derived from swap logs)
- `0x02` → WETH/ARB pool (`0xc6f78049...`) with WETH `0x82af...` and ARB `0x912c...`
- `0x08` → WETH/USDC pool (`0xc6962004...`)
- `0x12` → WETH/USDT pool (`0x641c00a8...`)
- `0x17` → WBTC/USDT pool (`0x5969efdd...`)
- `0x18` → WBTC/USDC pool (`0x0e483131...`)

### Likely bucket
**PackedUniV3Swap_ExactInput_or_MarketWithPriceLimit**  
**Confidence:** 0.72

**Evidence:**
- Direct UniV3 Swap events during successful txs.
- Custom packed payload with market ID and limit-like fields.
- Rough 50/50 success/failure rate suggests **conditional price triggers**.

**Caveat:** exact field meanings (minOut vs priceLimit) remain uncertain without source/ABI.

---

## Selector: `0x6bfd6286` (count 69)
**Destination contract:** same router `0x27920e80...`  
**Input length:** 68 bytes total (4 selector + 64 data)  
**Log count distribution:** 4 logs (13), 5 logs (8), 0 logs (48)  
**Additional event:** topic `0x40e9cecb...` from `0x360e68fa...` (router-level event).

### Calldata structure
- **Two 32-byte words**.
- **Word1:** same packed structure as `b2460c48` (market ID at byte[1], trailing 0x01 flag).
- **Word2:** small uint in the low 4 bytes (examples: `0x154a0eeb`, `0x11adc067`, `0x27c93473`).

### Observed markets
- Market IDs 3, 4, 5 map to WBTC/USDT, WETH/USDC, WBTC/USDC pools.

### Likely bucket
**PackedUniV3Swap_WithExtraLimitParam**  
**Confidence:** 0.68

**Evidence:**
- Same router and market selection logic as b2460c48.
- Additional 32-byte word likely encodes **minOut/limit price/trigger**.
- High revert rate (48/69) suggests **conditional execution**.

**Caveat:** cannot conclusively map Word2 to minOut vs price-limit without ABI/source.

---

## Selector: `0xb42ebb63` (count 33)
**Destination contract:** same router `0x27920e80...`  
**Input length:** 36 bytes total (4 selector + 32 packed data)  
**Log count distribution:** 4 logs (25), 0 logs (8)  

### Observed markets (from log addresses)
- Market 1 → WBTC/USDC pool (`0x843ac8dc...`)
- Market 2 → WETH/WBTC pool (`0x4bfc22a4...`)
- Market 5 → WETH/USDC pool (`0xd9e2a1a6...`)

### Likely bucket
**PackedUniV3Swap_AlternateSideOrExactOutput**  
**Confidence:** 0.66

**Evidence:**
- Same packed payload size/format as b2460c48 but different market map.
- UniV3 swap + transfer logs on success; revert otherwise.

**Caveat:** whether this is “exactOutput” or “sell/buy side” is inferred, not proven.

---

## Selector: `0x000000e7` (count 20)
**Destination contract:** `0x96daa0b8a5499ea9323421ed0cda06b345caab73`  
**Input length:** 164 bytes total (4 selector + 160 data)  
**Log count distribution:** 3 logs (16), 0 logs (4)

### Calldata structure (ABI-like)
5x 32-byte words:
1. **Pool address** (UniV3 ETH/USDC pool `0xc6962004...`)
2. **Direction flag** (`1` in all samples)
3. **Uint256 parameter** (varies, e.g., `0x812ee164ea6aed2c`)
4. **Uint256 parameter** (varies, ends in many zeros)
5. **Token address** (WETH `0x82af...`)

### Likely bucket
**SinglePoolSwap_CustomExecutor**  
**Confidence:** 0.55

**Evidence:**
- UniV3 swap event emitted by pool address; ERC20 transfers of WETH and USDC.
- Input includes pool address + direction + token address.

**Caveat:** exact param roles (amountIn/out vs sqrtPriceLimit) unresolved.

---

## ABI/Decompile Notes
- 4byte and OpenChain signature DBs return no matches for these selectors.
- The router bytecode shows explicit bit extraction (subroutine near 0x2b2b), confirming **custom packed encoding**.
- Without verified source code or internal ABI, field-level semantics remain partially inferred.

---

## Deliverable Status
**PARTIAL_WITH_BLOCKERS**

### Blockers
1. **Contract source/ABI not verified** for `0x27920e80...` and `0x96daa0b8...` (no Etherscan API key available to confirm).  
2. **Exact field mapping** (minOut vs priceLimit vs amountIn) requires ABI/source or deeper decompilation.

### Next Best Steps
- Obtain Arbiscan/Etherscan API key (or internal registry) to fetch verified source if available.
- Run a dedicated decompiler (panoramix/ethervm decompiler) to label storage variables and calldata fields.
- Simulate swaps with known inputs to map packed fields to price limits and amounts.
