# Retryable fee pricing, stated as a CuNi law

This directory states Arbitrum's retryable-ticket submission pricing as a
single [CuNi](https://github.com/ceedot-rock/cuni) law
(`retryable_fee.cuni`) and proves it byte-identical against the
implementation on a 34-vector battery (plus 16 refusal cases).

## What is proven

When a retryable ticket is submitted, ArbOS charges five exact-integer
(wei) computations. Each is stated once in the law, with the nitro source
lines it restates cited in comments:

| Law function | What it prices | Nitro source (v3.11.4) |
|---|---|---|
| `submission_fee` | L1-calldata component: `l1BaseFee * (1400 + 6*calldataLen)` | `arbos/retryables/retryable.go:394-398` |
| `submission_refund` | Excess of max submission fee over the fee, capped by the deposit | `arbos/tx_processor.go:374,377` via `takeFunds` at `:102-113` |
| `l2_gas_cost` | Execution gas: `effectiveBaseFee * gasLimit` | `arbos/tx_processor.go:454` |
| `infra_fee_cost` | ArbOS v11+ infra split: `min(gasCost, min(minBaseFee, baseFee) * gas)` | `arbos/tx_processor.go:462-464` |
| `gas_price_refund` | Excess of gas fee cap over base fee, capped by the deposit | `arbos/tx_processor.go:480,485` |

There is no division anywhere in the law, so there is no rounding
behavior to dispute. Every function is total on its stated domain and
refuses outside it (negative inputs, a max fee below the computed fee,
the gas-estimation-only negative branch).

## The kinks

Fee bugs live at boundaries, so the battery hits every kink from both
sides: zero-length calldata (the 1400-weight ticket overhead still
applies — the zero boundary is priced, not free), the
`min(minBaseFee, baseFee)` crossover in the infra split, and both refund
caps where the deposit binds instead of the excess.

## How it was verified

- `vectors.json`: 34 accept vectors generated from a standalone Go
  transcription of the formulas (`gen_vectors.go`, math/big only, each
  step citing its nitro source lines). `submission_fee` vectors were
  additionally checked against the real
  `retryables.RetryableSubmissionFee` in `retryable_fee_test.go`.
- `battery.py` / `battery.js` / `gobattery`: run every vector plus 16
  refusal cases through the CuNi law's Python, JavaScript, and Go seats.
  Result: **50/50 on all three seats** — every value byte-identical,
  every refusal firing with its named reason.
- `cuni check retryable_fee.cuni`: exactness PASS across the CuNi seat
  catalog.

## Domain honesty

The law refuses inputs whose fee would exceed 2^53 - 1. That bound
exists so the JavaScript seat (f64 numbers) stays exact; the Go seat
wraps int64 silently and Python is unbounded, so rather than let seats
disagree, the law refuses. All realistic mainnet values sit deep inside
the domain. The one realistic-looking case that exceeds it (100 gwei
base fee × 5M gas = 5e17 wei) is documented as out of domain, not
silently approximated.

## Files

- `retryable_fee.cuni` — the law
- `vectors.json` — the 34-vector battery (args and expected values)
- `testdata/vectors.json` — same battery for the Go test
- `retryable_fee_test.go` — Go test running the `submission_fee`
  vectors against the real implementation
- `gen_vectors.go` — the standalone vector generator (reference only)
