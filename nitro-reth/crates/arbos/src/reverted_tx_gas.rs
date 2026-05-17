//! Hardcoded per-tx gas overrides for previously-reverted transactions.
//!
//! Mirrors Nitro's `go-ethereum/core/reverted_tx_gas.go`: when a specific
//! transaction hash appears in this table, `RevertedTxHook` short-circuits
//! execution and forces the recorded `l2GasUsed` value. Used to repair
//! historical Sepolia divergence caused by a Stylus bug; replays must apply
//! the same override or block hashes diverge.

use alloy_primitives::{b256, B256};

/// Lookup the recorded L2 gas-used for a tx hash. `Some(g)` means the tx
/// must be force-reverted with `g` total L2 gas (excludes poster gas).
pub fn lookup(tx_hash: B256) -> Option<u64> {
    // Arbitrum Sepolia (chain_id=421614). Tx timestamp: Oct-13-2025 03:30:36 AM +UTC.
    const SEPOLIA_STYLUS_INCIDENT: B256 =
        b256!("58df300a7f04fe31d41d24672786cbe1c58b4f3d8329d0d74392d814dd9f7e40");
    if tx_hash == SEPOLIA_STYLUS_INCIDENT {
        return Some(45_174);
    }
    None
}
