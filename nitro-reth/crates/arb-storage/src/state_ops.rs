use alloy_primitives::{Address, Bytes, U256, address, keccak256};
use revm::Database;
use std::collections::HashMap;

/// ArbOS state address — the fictional account that stores all ArbOS state.
pub const ARBOS_STATE_ADDRESS: Address = address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");

/// Filtered transactions state address — a separate account for tracking filtered tx hashes.
pub const FILTERED_TX_STATE_ADDRESS: Address =
    address!("a4b0500000000000000000000000000000000001");

/// Ensures the ArbOS account exists in bundle_state.
///
/// Uses database.basic() instead of state.basic() to avoid cache non-determinism.
pub fn ensure_arbos_account_in_bundle<D: Database>(state: &mut revm::database::State<D>) {
    ensure_account_in_bundle(state, ARBOS_STATE_ADDRESS);
}

/// Ensures an arbitrary account exists in bundle_state with nonce=1.
pub fn ensure_account_in_bundle<D: Database>(
    state: &mut revm::database::State<D>,
    addr: Address,
) {
    use revm_database::{AccountStatus, BundleAccount};
    use revm_state::AccountInfo;

    if state.bundle_state.state.contains_key(&addr) {
        return;
    }

    let db_info = state.database.basic(addr).ok().flatten();

    let info = db_info.or_else(|| {
        Some(AccountInfo {
            balance: U256::ZERO,
            nonce: 1,
            code_hash: keccak256([]),
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
    state.bundle_state.state.insert(addr, acc);
}

/// Ensures the account exists in the cache. If the account doesn't exist
/// (database returned None), creates it with default values (nonce=0, balance=0).
fn ensure_cache_account<D: Database>(state: &mut revm::database::State<D>, addr: Address) {
    use revm_database::AccountStatus;

    let _ = state.load_cache_account(addr);

    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        if cached.account.is_none() {
            cached.account = Some(revm_database::PlainAccount {
                info: revm_state::AccountInfo {
                    balance: U256::ZERO,
                    nonce: 0,
                    code_hash: keccak256([]),
                    code: None,
                    account_id: None,
                },
                storage: Default::default(),
            });
            cached.status = AccountStatus::InMemoryChange;
        }
    }
}

/// Reads a storage slot from the ArbOS account, checking cache -> bundle -> database.
pub fn read_arbos_storage<D: Database>(
    state: &mut revm::database::State<D>,
    slot: U256,
) -> U256 {
    read_storage_at(state, ARBOS_STATE_ADDRESS, slot)
}

/// Reads a storage slot from an arbitrary account, checking cache -> bundle -> database.
pub fn read_storage_at<D: Database>(
    state: &mut revm::database::State<D>,
    account: Address,
    slot: U256,
) -> U256 {
    // Check cache first
    if let Some(cached_acc) = state.cache.accounts.get(&account) {
        if let Some(ref account) = cached_acc.account {
            if let Some(&value) = account.storage.get(&slot) {
                return value;
            }
        }
    }

    // Check bundle_state
    if let Some(acc) = state.bundle_state.state.get(&account) {
        if let Some(slot_entry) = acc.storage.get(&slot) {
            return slot_entry.present_value;
        }
    }

    // Fall back to database
    state
        .database
        .storage(account, slot)
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
    write_storage_at(state, ARBOS_STATE_ADDRESS, slot, value);
}

/// Writes a storage slot to an arbitrary account using the transition mechanism.
pub fn write_storage_at<D: Database>(
    state: &mut revm::database::State<D>,
    account: Address,
    slot: U256,
    value: U256,
) {
    use revm_database::states::StorageSlot;

    // Ensure account exists in cache (creates it if database returns None).
    ensure_cache_account(state, account);

    // Get current value from cache/bundle, and original from DB
    let current_value = {
        state
            .cache
            .accounts
            .get(&account)
            .and_then(|ca| ca.account.as_ref())
            .and_then(|a| a.storage.get(&slot).copied())
    }
    .or_else(|| {
        state
            .bundle_state
            .state
            .get(&account)
            .and_then(|a| a.storage.get(&slot))
            .map(|s| s.present_value)
    });

    let original_value = state
        .database
        .storage(account, slot)
        .unwrap_or(U256::ZERO);

    // Skip no-op writes
    let prev_value = current_value.unwrap_or(original_value);
    if value == prev_value {
        return;
    }

    tracing::info!(
        target: "arb::storage_write",
        %account,
        slot = %slot,
        prev = %prev_value,
        new = %value,
        original_db = %original_value,
        "write_storage_at"
    );

    // Modify cache entry
    let (previous_info, previous_status, current_info, current_status) = {
        let cached_acc = match state.cache.accounts.get_mut(&account) {
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

    state.apply_transition(vec![(account, transition)]);
}

/// Reads the balance of an account from the state.
pub fn get_account_balance<D: Database>(
    state: &mut revm::database::State<D>,
    addr: Address,
) -> U256 {
    if let Some(cached_acc) = state.cache.accounts.get(&addr) {
        if let Some(ref account) = cached_acc.account {
            return account.info.balance;
        }
    }

    state
        .database
        .basic(addr)
        .ok()
        .flatten()
        .map(|info| info.balance)
        .unwrap_or(U256::ZERO)
}

/// Sets the nonce of an account, loading it into cache if needed.
pub fn set_account_nonce<D: Database>(
    state: &mut revm::database::State<D>,
    addr: Address,
    nonce: u64,
) {
    ensure_cache_account(state, addr);

    let (previous_info, previous_status, current_info, current_status) = {
        let cached_acc = match state.cache.accounts.get_mut(&addr) {
            Some(acc) => acc,
            None => return,
        };
        let previous_status = cached_acc.status;
        let previous_info = cached_acc.account.as_ref().map(|a| a.info.clone());

        if let Some(ref mut account) = cached_acc.account {
            account.info.nonce = nonce;
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

    let transition = revm::database::TransitionAccount {
        info: current_info,
        status: current_status,
        previous_info,
        previous_status,
        storage: HashMap::default(),
        storage_was_destroyed: false,
    };
    state.apply_transition(vec![(addr, transition)]);
}

/// Sets the code of an account, loading it into cache if needed.
pub fn set_account_code<D: Database>(
    state: &mut revm::database::State<D>,
    addr: Address,
    code: Bytes,
) {
    use revm_state::Bytecode;

    ensure_cache_account(state, addr);
    let code_hash = keccak256(&code);
    let bytecode = Bytecode::new_raw(code);

    let (previous_info, previous_status, current_info, current_status) = {
        let cached_acc = match state.cache.accounts.get_mut(&addr) {
            Some(acc) => acc,
            None => return,
        };
        let previous_status = cached_acc.status;
        let previous_info = cached_acc.account.as_ref().map(|a| a.info.clone());

        if let Some(ref mut account) = cached_acc.account {
            account.info.code_hash = code_hash;
            account.info.code = Some(bytecode);
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

    let transition = revm::database::TransitionAccount {
        info: current_info,
        status: current_status,
        previous_info,
        previous_status,
        storage: HashMap::default(),
        storage_was_destroyed: false,
    };
    state.apply_transition(vec![(addr, transition)]);
}
