//! `eth_sendRawTransactionConditional` — Arbitrum's conditional tx
//! submission RPC.
//!
//! Lets a client attach predicates (block-number range, timestamp
//! range, per-account storage roots / slot values) to a raw tx. The
//! sequencer only accepts the tx if every predicate holds against the
//! current chain state at submission time. Used by MEV-aware clients
//! to fail fast when a trade opportunity has already been consumed.
//!
//! Matches Nitro's `arbitrum_types.ConditionalOptions` +
//! `SubmitConditionalTransaction` in
//! `/go-ethereum/arbitrum/conditionaltx.go`.

use std::collections::HashMap;

use alloy_primitives::{Address, Bytes, B256};
use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    types::{error::INVALID_PARAMS_CODE, ErrorObject},
};
use serde::{Deserialize, Serialize};

/// Per-account expected state:
///   - Either an expected storage-root hash
///   - Or a map of slot → expected value
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(untagged)]
pub enum KnownAccountCondition {
    /// Entire storage root must match.
    RootHash(B256),
    /// Specific storage slots must have the given values.
    #[serde(rename_all = "camelCase")]
    SlotValues(HashMap<B256, B256>),
    #[default]
    Empty,
}

/// Conditional options attached to a raw tx.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ConditionalOptions {
    /// Per-account storage-root or slot-value requirements.
    #[serde(default, skip_serializing_if = "HashMap::is_empty")]
    pub known_accounts: HashMap<Address, KnownAccountCondition>,
    /// L1 block number must be ≥ this.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub block_number_min: Option<u64>,
    /// L1 block number must be ≤ this.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub block_number_max: Option<u64>,
    /// L2 block timestamp must be ≥ this.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub timestamp_min: Option<u64>,
    /// L2 block timestamp must be ≤ this.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub timestamp_max: Option<u64>,
}

fn condition_rejected(reason: &str) -> ErrorObject<'static> {
    ErrorObject::owned(
        INVALID_PARAMS_CODE,
        format!("conditional tx rejected: {reason}"),
        None::<()>,
    )
}

/// Check `(block_number_*, timestamp_*)` predicates against current
/// chain state. Returns on the first predicate that fails.
///
/// Per-account storage checks are handled separately since they need
/// provider access.
pub fn check_simple_predicates(
    opts: &ConditionalOptions,
    current_l1_block: u64,
    current_l2_timestamp: u64,
) -> Result<(), ErrorObject<'static>> {
    if let Some(min) = opts.block_number_min {
        if current_l1_block < min {
            return Err(condition_rejected("BlockNumberMin condition not met"));
        }
    }
    if let Some(max) = opts.block_number_max {
        if current_l1_block > max {
            return Err(condition_rejected("BlockNumberMax condition not met"));
        }
    }
    if let Some(min) = opts.timestamp_min {
        if current_l2_timestamp < min {
            return Err(condition_rejected("TimestampMin condition not met"));
        }
    }
    if let Some(max) = opts.timestamp_max {
        if current_l2_timestamp > max {
            return Err(condition_rejected("TimestampMax condition not met"));
        }
    }
    Ok(())
}

/// `eth_sendRawTransactionConditional` — registered on the `eth`
/// namespace in Nitro (not `arb_`). We expose it here and let the
/// node-level RPC module merger handle namespace binding.
#[rpc(server, namespace = "eth")]
pub trait ConditionalTxApi {
    /// Submit a signed raw tx with attached predicates. Returns the
    /// tx hash on acceptance; error on predicate failure or pool
    /// rejection.
    #[method(name = "sendRawTransactionConditional")]
    async fn send_raw_transaction_conditional(
        &self,
        raw_tx: Bytes,
        options: ConditionalOptions,
    ) -> RpcResult<B256>;
}

#[cfg(test)]
mod tests {
    use super::*;

    fn some_opts() -> ConditionalOptions {
        ConditionalOptions {
            block_number_min: Some(100),
            block_number_max: Some(200),
            timestamp_min: Some(1_700_000_000),
            timestamp_max: Some(1_800_000_000),
            known_accounts: HashMap::new(),
        }
    }

    #[test]
    fn none_all_accepts() {
        let opts = ConditionalOptions::default();
        assert!(check_simple_predicates(&opts, 0, 0).is_ok());
    }

    #[test]
    fn block_number_min_rejects_below() {
        let opts = some_opts();
        let err = check_simple_predicates(&opts, 99, 1_750_000_000).unwrap_err();
        assert!(err.message().contains("BlockNumberMin"));
    }

    #[test]
    fn block_number_max_rejects_above() {
        let opts = some_opts();
        let err = check_simple_predicates(&opts, 201, 1_750_000_000).unwrap_err();
        assert!(err.message().contains("BlockNumberMax"));
    }

    #[test]
    fn timestamp_min_rejects_below() {
        let opts = some_opts();
        let err = check_simple_predicates(&opts, 150, 1_000).unwrap_err();
        assert!(err.message().contains("TimestampMin"));
    }

    #[test]
    fn timestamp_max_rejects_above() {
        let opts = some_opts();
        let err = check_simple_predicates(&opts, 150, 2_000_000_000).unwrap_err();
        assert!(err.message().contains("TimestampMax"));
    }

    #[test]
    fn inside_window_accepts() {
        let opts = some_opts();
        assert!(check_simple_predicates(&opts, 150, 1_750_000_000).is_ok());
    }

    #[test]
    fn boundary_inclusive() {
        let opts = some_opts();
        assert!(check_simple_predicates(&opts, 100, 1_700_000_000).is_ok());
        assert!(check_simple_predicates(&opts, 200, 1_800_000_000).is_ok());
    }
}
