//! Block production types and helpers.

use alloy_consensus::transaction::SignerRecoverable;
use alloy_evm::block::BlockExecutor;
use alloy_primitives::{Address, B256, Bytes, Signature, U256};
use arb_primitives::{
    signed_tx::{ArbTransactionSigned, ArbTypedTransaction},
    tx_types::ArbInternalTx,
};
use revm::{Database, database::BundleState};
use revm_database::{
    AccountStatus, BundleAccount, CacheState, StorageWithOriginalValues,
    states::plain_account::StorageSlot,
};
use rustc_hash::FxHashSet;
use tracing::debug;

/// Result of producing a block.
#[derive(Debug, Clone)]
pub struct ProducedBlock {
    /// Hash of the produced block.
    pub block_hash: B256,
    /// Send root from the block's extra_data.
    pub send_root: B256,
}

/// Error type for block production.
#[derive(Debug, thiserror::Error)]
pub enum BlockProducerError {
    #[error("state access: {0}")]
    StateAccess(String),
    #[error("execution: {0}")]
    Execution(String),
    #[error("storage: {0}")]
    Storage(String),
    #[error("parse: {0}")]
    Parse(String),
    #[error("unexpected: {0}")]
    Unexpected(String),
}

/// Create an internal transaction (type 0x6A).
pub fn create_internal_tx(chain_id: u64, data: &[u8]) -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(chain_id),
            data: Bytes::copy_from_slice(data),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    )
}

/// Execute and commit an internal transaction via the block executor.
pub fn execute_and_commit_tx<E>(
    executor: &mut E,
    tx: &ArbTransactionSigned,
    label: &str,
) -> Result<(), BlockProducerError>
where
    E: BlockExecutor<Transaction = ArbTransactionSigned>,
{
    let recovered = tx
        .clone()
        .try_into_recovered()
        .map_err(|e| BlockProducerError::Execution(format!("{label} recovery: {e}")))?;

    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| BlockProducerError::Execution(format!("{label} execution: {e}")))?;

    executor
        .commit_transaction(result)
        .map_err(|e| BlockProducerError::Execution(format!("{label} commit: {e}")))?;

    Ok(())
}

/// EIP-161: mark empty non-zombie accounts for trie deletion.
/// `db` must read parent-block state, not the in-flight `State` overlay.
pub fn delete_empty_accounts<DB: Database>(
    bundle: &mut BundleState,
    zombie_accounts: &FxHashSet<Address>,
    db: &mut DB,
) -> Result<(), BlockProducerError>
where
    DB::Error: core::fmt::Display,
{
    let keccak_empty = B256::from(alloy_primitives::keccak256([]));
    let mut to_remove = Vec::new();
    for (addr, account) in bundle.state.iter_mut() {
        if let Some(ref info) = account.info {
            let is_empty =
                info.nonce == 0 && info.balance.is_zero() && info.code_hash == keccak_empty;
            if is_empty && !zombie_accounts.contains(addr) {
                let existed_before = db
                    .basic(*addr)
                    .map_err(|e| BlockProducerError::Storage(e.to_string()))?
                    .is_some();
                if existed_before {
                    debug!(
                        target: "block_producer",
                        addr = ?addr,
                        "EIP-161: deleting empty account from state"
                    );
                    account.info = None;
                } else {
                    to_remove.push(*addr);
                }
            }
        }
    }
    for addr in to_remove {
        bundle.state.remove(&addr);
    }
    Ok(())
}

/// Remove unchanged storage slots from the bundle.
pub fn filter_unchanged_storage(bundle: &mut BundleState) {
    for (_addr, account) in bundle.state.iter_mut() {
        account
            .storage
            .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
    }
}

/// Augment the bundle with direct cache modifications not captured by EVM transitions.
/// `db` must read parent-block state, not the in-flight `State` overlay.
pub fn augment_bundle_from_cache<DB: Database>(
    bundle: &mut BundleState,
    cache: &CacheState,
    db: &mut DB,
) -> Result<(), BlockProducerError>
where
    DB::Error: core::fmt::Display,
{
    for (addr, cache_acct) in &cache.accounts {
        let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
        let current_storage = cache_acct
            .account
            .as_ref()
            .map(|a| &a.storage)
            .cloned()
            .unwrap_or_default();

        if let Some(bundle_acct) = bundle.state.get_mut(addr) {
            // Update existing bundle entry from cache.
            bundle_acct.info = current_info;

            for (key, value) in &current_storage {
                if let Some(slot) = bundle_acct.storage.get_mut(key) {
                    slot.present_value = *value;
                } else {
                    // Slot written via direct cache modification.
                    let original_value = db
                        .storage(*addr, *key)
                        .map_err(|e| BlockProducerError::Storage(e.to_string()))?;
                    if *value != original_value {
                        bundle_acct.storage.insert(
                            *key,
                            StorageSlot {
                                previous_or_original_value: original_value,
                                present_value: *value,
                            },
                        );
                    }
                }
            }
        } else {
            // Account not in bundle — check if modified from original.
            let original_info = db
                .basic(*addr)
                .map_err(|e| BlockProducerError::Storage(e.to_string()))?;

            let info_changed = match (&original_info, &current_info) {
                (None, None) => false,
                (Some(_), None) | (None, Some(_)) => true,
                (Some(original), Some(current)) => {
                    original.balance != current.balance
                        || original.nonce != current.nonce
                        || original.code_hash != current.code_hash
                }
            };

            let mut storage_changes: StorageWithOriginalValues = Default::default();
            for (key, value) in &current_storage {
                let original_value = db
                    .storage(*addr, *key)
                    .map_err(|e| BlockProducerError::Storage(e.to_string()))?;
                if original_value != *value {
                    storage_changes.insert(
                        *key,
                        StorageSlot {
                            previous_or_original_value: original_value,
                            present_value: *value,
                        },
                    );
                }
            }

            if info_changed || !storage_changes.is_empty() {
                let status = if original_info.is_some() {
                    AccountStatus::Changed
                } else {
                    AccountStatus::InMemoryChange
                };

                bundle.state.insert(
                    *addr,
                    BundleAccount {
                        info: current_info,
                        original_info,
                        storage: storage_changes,
                        status,
                    },
                );
            }
        }
    }
    Ok(())
}
