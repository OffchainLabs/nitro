# Fuzz targets

Run with `cargo +nightly fuzz run <target>`. Requires `cargo install cargo-fuzz`.

## Targets

| Target | Surface |
|---|---|
| `parse_l2_transactions` | L1→L2 message parser (all message kinds) |
| `decode_arb_receipt` | EIP-2718 ArbReceipt decode |
| `decode_arb_tx_envelope` | ArbTxEnvelope (all tx variants) decode |
| `decode_internal_tx` | startBlock internal-tx ABI decode |

## Examples

```bash
# Run for 60 seconds
cargo +nightly fuzz run parse_l2_transactions -- -max_total_time=60

# With a corpus directory
cargo +nightly fuzz run parse_l2_transactions corpus/parse_l2_transactions

# Reproduce a regression
cargo +nightly fuzz run parse_l2_transactions corpus/parse_l2_transactions/<crash-input>
```

The fuzz workspace is independent of the main workspace; nothing here links into the node binary.
