# TB3 Transactions with Signatures & Inferred Conditionals

Full machine table: `TB3-TX-CONDITIONALS-TABLE.csv` (381 rows).

## Selector summary

- `0xb2460c48`: 232 txs | bucket: **PackedUniV3Swap_ExactInput_or_MarketWithPriceLimit** | confidence: 0.72
- `0x6bfd6286`: 69 txs | bucket: **PackedUniV3Swap_WithExtraLimitParam** | confidence: 0.68
- `0xb42ebb63`: 33 txs | bucket: **PackedUniV3Swap_AlternateSideOrExactOutput** | confidence: 0.66
- `0x000000e7`: 20 txs | bucket: **SinglePoolSwap_CustomExecutor** | confidence: 0.55
- `0x00000000`: 11 txs | bucket: **Unknown** | confidence: n/a
- `0xa28dd96e`: 9 txs | bucket: **Unknown** | confidence: n/a
- `0x2f2a2543`: 3 txs | bucket: **Unknown** | confidence: n/a
- `0x01000000`: 2 txs | bucket: **Unknown** | confidence: n/a
- `0x82af4944`: 1 txs | bucket: **Unknown** | confidence: n/a
- `0x00003802`: 1 txs | bucket: **Unknown** | confidence: n/a

## Exemplar tx table

| txHash | selector | status | to | inferred conditionals | confidence |
|---|---|---|---|---|---|
| `0x7b975b0035…` | `0xb2460c48` | success | `0x27920e80…` | packed order: marketId + amount cap + price/limit field + flag; frequent conditional reverts | 0.72 |
| `0x9799757a8e…` | `0xb2460c48` | success | `0x27920e80…` | packed order: marketId + amount cap + price/limit field + flag; frequent conditional reverts | 0.72 |
| `0x6177376359…` | `0xb2460c48` | success | `0x27920e80…` | packed order: marketId + amount cap + price/limit field + flag; frequent conditional reverts | 0.72 |
| `0x5beb952282…` | `0xb2460c48` | revert | `0x27920e80…` | packed order: marketId + amount cap + price/limit field + flag; frequent conditional reverts | 0.72 |
| `0x6debf8d8be…` | `0x6bfd6286` | success | `0x27920e80…` | packed order: marketId + maxAmountIn + minAmountOut (word1 low4) + price/limit field | 0.68 |
| `0x48df1973e1…` | `0x6bfd6286` | success | `0x27920e80…` | packed order: marketId + maxAmountIn + minAmountOut (word1 low4) + price/limit field | 0.68 |
| `0xb23ca6d7e3…` | `0x6bfd6286` | success | `0x27920e80…` | packed order: marketId + maxAmountIn + minAmountOut (word1 low4) + price/limit field | 0.68 |
| `0xee2784c5ef…` | `0x6bfd6286` | revert | `0x27920e80…` | packed order: marketId + maxAmountIn + minAmountOut (word1 low4) + price/limit field | 0.68 |
| `0x9ad535e34f…` | `0xb42ebb63` | success | `0x27920e80…` | packed order variant: marketId + amount field + limit/side field + flag | 0.66 |
| `0x537141e757…` | `0xb42ebb63` | success | `0x27920e80…` | packed order variant: marketId + amount field + limit/side field + flag | 0.66 |
| `0x44257d7b3f…` | `0xb42ebb63` | success | `0x27920e80…` | packed order variant: marketId + amount field + limit/side field + flag | 0.66 |
| `0xa213bf623c…` | `0xb42ebb63` | revert | `0x27920e80…` | packed order variant: marketId + amount field + limit/side field + flag | 0.66 |
| `0x8ee9deabb9…` | `0x000000e7` | revert | `0x96daa0b8…` | ABI-like 5-word executor: pool addr + direction + amount/limit params + token | 0.55 |
| `0x852649ef65…` | `0x000000e7` | success | `0x96daa0b8…` | ABI-like 5-word executor: pool addr + direction + amount/limit params + token | 0.55 |
| `0x6a980cd34b…` | `0x000000e7` | success | `0x96daa0b8…` | ABI-like 5-word executor: pool addr + direction + amount/limit params + token | 0.55 |

## Notes
- This table is tx-complete and signature-complete for the sampled true-flow corpus.
- Field semantics are inferred from clustering + traces; exact byte offsets are highest confidence for `0x6bfd6286` and partial for others.
