//! Hardcoded per-tx gas overrides for previously-reverted transactions.
//!
//! When a tx hash appears in this table, `RevertedTxHook` short-circuits
//! execution and forces the recorded `l2GasUsed`. Used to repair historical
//! Sepolia divergence from a Stylus ARM/x86 determinism incident; replays
//! must apply the same override or block hashes diverge.

use alloy_primitives::{b256, B256};

/// Lookup the recorded L2 gas-used for a tx hash. `Some(g)` means the tx
/// must be force-reverted with `g` total L2 gas (excludes poster gas).
///
/// Both entries stem from the same Sepolia ARM-vs-x86 Stylus determinism
/// incident at block ~204,060,000: a contract activated at v40 with the
/// default MaxStackDepth recurses deeply enough that Cranelift's stack
/// frames consume Rust call stack differently on ARM vs x86, so the two
/// architectures terminate at different ink levels and report different gas.
/// arbreth runs on arm64, so this override is required for replay parity.
pub fn lookup(tx_hash: B256) -> Option<u64> {
    // tx 0x58df300a — block 204,060,366. Canon: gasUsed=0xb226=45_606,
    // l2-only (canon - 432 calldata) = 45_174.
    const SEPOLIA_STYLUS_INCIDENT_A: B256 =
        b256!("58df300a7f04fe31d41d24672786cbe1c58b4f3d8329d0d74392d814dd9f7e40");
    // tx 0xe22b6570 — block 204,060,502. Same canon shape (45_606 → 45_174).
    const SEPOLIA_STYLUS_INCIDENT_B: B256 =
        b256!("e22b6570bd5e539adb0363602edfc2ceeb979802d7697dcd3b203d2d734176da");
    if tx_hash == SEPOLIA_STYLUS_INCIDENT_A || tx_hash == SEPOLIA_STYLUS_INCIDENT_B {
        return Some(45_174);
    }
    None
}
