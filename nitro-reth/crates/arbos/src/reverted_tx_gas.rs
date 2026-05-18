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
///
/// Both entries below stem from the same Oct-13-2025 Sepolia ARM-vs-x86
/// Stylus determinism incident: the contract at
/// `0x68c709da6c89bb74501530f2b9d0970b9a08165a` was activated at block
/// 204,059,808 with the ArbOS v40 default `MaxStackDepth = 262_144`.
/// Calls into it recurse deeply enough that Cranelift's compiled stack
/// frames consume Rust call stack differently on ARM vs x86, so the two
/// architectures terminate the recursion at different ink levels and
/// report different gas. Nitro hardcoded the first occurrence
/// (`0x58df300a…`, block 204,060,366) to prevent chain divergence — see
/// `nitro/go-ethereum/core/reverted_tx_gas.go` and the original commit
/// message "*bypass transaction execution for problematic txs execution
/// on ARM architecture*". The second occurrence (`0xe22b6570…`, block
/// 204,060,502, 34 s later, same caller / contract / selector) was *not*
/// added to Nitro's table — probably oversight, since the same ARM
/// divergence applies. arbreth runs on arm64 (Apple Silicon), so without
/// this entry the second tx will keep producing +142 gas vs canon.
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
