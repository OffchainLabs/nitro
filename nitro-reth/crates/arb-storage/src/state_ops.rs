use alloy_primitives::{Address, U256, address};
use revm::Database;
use std::collections::HashMap;

/// ArbOS state address — the fictional account that stores all ArbOS state.
pub const ARBOS_STATE_ADDRESS: Address = address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");

/// Ensures the ArbOS account exists in bundle_state.
///
/// Uses database.basic() instead of state.basic() to avoid cache non-determinism.
pub fn ensure_arbos_account_in_bundle<D: Database>(state: &mut revm::database::State<D>) {
    use revm_database::{AccountStatus, BundleAccount};
    use revm_state::AccountInfo;

    if state.bundle_state.state.contains_key(&ARBOS_STATE_ADDRESS) {
        return;
    }

    let db_info = state
        .database
        .basic(ARBOS_STATE_ADDRESS)
        .ok()
        .flatten();

    let info = db_info.or_else(|| {
        Some(AccountInfo {
            balance: U256::ZERO,
            nonce: 1,
            code_hash: alloy_primitives::keccak256([]),
            code: None,
            account_id: None,
        })
    });

    let acc = BundleAccount {
        info: info.clone(),
        storage: HashMap::default(),
        original_info: info,
        status: AccountStatus::Loaded,
    };
    state.bundle_state.state.insert(ARBOS_STATE_ADDRESS, acc);
}

/// Reads a storage slot from the ArbOS account, checking cache -> bundle -> database.
pub fn read_arbos_storage<D: Database>(
    state: &mut revm::database::State<D>,
    slot: U256,
) -> U256 {
    // Check cache first
    if let Some(cached_acc) = state.cache.accounts.get(&ARBOS_STATE_ADDRESS) {
        if let Some(ref account) = cached_acc.account {
            if let Some(&value) = account.storage.get(&slot) {
                return value;
            }
        }
    }

    // Check bundle_state
    if let Some(acc) = state.bundle_state.state.get(&ARBOS_STATE_ADDRESS) {
        if let Some(slot_entry) = acc.storage.get(&slot) {
            return slot_entry.present_value;
        }
    }

    // Fall back to database
    state
        .database
        .storage(ARBOS_STATE_ADDRESS, slot)
        .unwrap_or(U256::ZERO)
}

/// Writes a storage slot to the ArbOS account using the transition mechanism.
///
/// This ensures changes survive merge_transitions() and are properly journaled.
/// Skips no-op writes where value == current value.
pub fn write_arbos_storage<D: Database>(
    state: &mut revm::database::State<D>,
    slot: U256,
    value: U256,
) {
    use revm_database::states::StorageSlot;

    // Load account into cache
    let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);

    // Get current value from cache/bundle, and original from DB
    let current_value = {
        state
            .cache
            .accounts
            .get(&ARBOS_STATE_ADDRESS)
            .and_then(|ca| ca.account.as_ref())
            .and_then(|a| a.storage.get(&slot).copied())
    }
    .or_else(|| {
        state
            .bundle_state
            .state
            .get(&ARBOS_STATE_ADDRESS)
            .and_then(|a| a.storage.get(&slot))
            .map(|s| s.present_value)
    });

    let original_value = state
        .database
        .storage(ARBOS_STATE_ADDRESS, slot)
        .unwrap_or(U256::ZERO);

    // Skip no-op writes
    let prev_value = current_value.unwrap_or(original_value);
    if value == prev_value {
        return;
    }

    // Modify cache entry
    let (previous_info, previous_status, current_info, current_status) = {
        let cached_acc = match state.cache.accounts.get_mut(&ARBOS_STATE_ADDRESS) {
            Some(acc) => acc,
            None => return,
        };

        let previous_status = cached_acc.status;
        let previous_info = cached_acc.account.as_ref().map(|a| a.info.clone());

        if let Some(ref mut account) = cached_acc.account {
            account.storage.insert(slot, value);
        }

        let had_no_nonce_and_code = previous_info
            .as_ref()
            .map(|info| info.has_no_code_and_nonce())
            .unwrap_or_default();
        cached_acc.status = cached_acc.status.on_changed(had_no_nonce_and_code);

        let current_info = cached_acc.account.as_ref().map(|a| a.info.clone());
        let current_status = cached_acc.status;
        (previous_info, previous_status, current_info, current_status)
    };

    // Create and apply transition
    let mut storage_changes: revm_database::StorageWithOriginalValues = HashMap::default();
    storage_changes.insert(slot, StorageSlot::new_changed(original_value, value));

    let transition = revm::database::TransitionAccount {
        info: current_info,
        status: current_status,
        previous_info,
        previous_status,
        storage: storage_changes,
        storage_was_destroyed: false,
    };

    state.apply_transition(vec![(ARBOS_STATE_ADDRESS, transition)]);
}
