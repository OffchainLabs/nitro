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
        // Log no-op writes for the gasBacklog slot (last byte 0x04 in L2 pricing subspace)
        if account == ARBOS_STATE_ADDRESS {
            tracing::debug!(
                target: "arb::storage",
                ?slot,
                ?value,
                ?prev_value,
                from_cache = current_value.is_some(),
                "write_storage_at no-op (value==prev)"
            );
        }
        return;
    }

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
    if account == ARBOS_STATE_ADDRESS {
        tracing::debug!(
            target: "arb::storage",
            ?slot,
            ?value,
            ?prev_value,
            ?original_value,
            "write_storage_at applying transition"
        );
    }
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

#[cfg(test)]
mod tests {
    use super::*;
    use revm_database::states::bundle_state::BundleRetention;
    use revm_database::StateBuilder;

    /// In-memory database that returns empty for everything.
    #[derive(Default)]
    struct EmptyDb;

    impl Database for EmptyDb {
        type Error = std::convert::Infallible;
        fn basic(
            &mut self,
            _address: Address,
        ) -> Result<Option<revm_state::AccountInfo>, Self::Error> {
            Ok(None)
        }
        fn code_by_hash(
            &mut self,
            _code_hash: alloy_primitives::B256,
        ) -> Result<revm_state::Bytecode, Self::Error> {
            Ok(revm_state::Bytecode::default())
        }
        fn storage(
            &mut self,
            _address: Address,
            _index: U256,
        ) -> Result<U256, Self::Error> {
            Ok(U256::ZERO)
        }
        fn block_hash(
            &mut self,
            _number: u64,
        ) -> Result<alloy_primitives::B256, Self::Error> {
            Ok(alloy_primitives::B256::ZERO)
        }
    }

    fn make_state() -> revm::database::State<EmptyDb> {
        StateBuilder::new()
            .with_database(EmptyDb)
            .with_bundle_update()
            .build()
    }

    #[test]
    fn test_write_storage_at_creates_transition() {
        let mut state = make_state();
        let slot = U256::from(42);
        let value = U256::from(12345);

        write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot, value);

        // Verify value is in cache.
        let cached = state
            .cache
            .accounts
            .get(&ARBOS_STATE_ADDRESS)
            .unwrap();
        let stored = cached
            .account
            .as_ref()
            .unwrap()
            .storage
            .get(&slot)
            .copied()
            .unwrap();
        assert_eq!(stored, value, "Value should be in cache");

        // Merge transitions into bundle.
        state.merge_transitions(BundleRetention::Reverts);
        let bundle = state.take_bundle();

        // Verify value is in bundle.
        let bundle_acct = bundle.state.get(&ARBOS_STATE_ADDRESS)
            .expect("ArbOS account should be in bundle after merge");
        let bundle_slot = bundle_acct.storage.get(&slot)
            .expect("Slot should be in bundle storage");
        assert_eq!(bundle_slot.present_value, value, "Bundle present_value should match");
    }

    #[test]
    fn test_write_zero_value_is_noop_for_new_slot() {
        let mut state = make_state();
        let slot = U256::from(42);

        // Writing 0 to a slot that doesn't exist (DB returns 0) should be a no-op.
        write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot, U256::ZERO);

        // After merge, the slot should NOT be in the bundle.
        state.merge_transitions(BundleRetention::Reverts);
        let bundle = state.take_bundle();

        // Account might or might not be in bundle, but the slot should not.
        if let Some(acct) = bundle.state.get(&ARBOS_STATE_ADDRESS) {
            assert!(
                acct.storage.get(&slot).is_none(),
                "Slot written with zero should not appear in bundle"
            );
        }
    }

    #[test]
    fn test_write_survives_multiple_transitions() {
        let mut state = make_state();
        let slot_a = U256::from(10);
        let slot_b = U256::from(20);

        // First transition: write slot A (simulates baseFee write during StartBlock).
        write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot_a, U256::from(100));

        // Second transition: write slot B (simulates gasBacklog write during user tx).
        write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot_b, U256::from(200));

        // Merge and check both survive.
        state.merge_transitions(BundleRetention::Reverts);
        let bundle = state.take_bundle();

        let acct = bundle.state.get(&ARBOS_STATE_ADDRESS)
            .expect("ArbOS account should be in bundle");
        assert_eq!(
            acct.storage.get(&slot_a).unwrap().present_value,
            U256::from(100),
            "Slot A should survive merge"
        );
        assert_eq!(
            acct.storage.get(&slot_b).unwrap().present_value,
            U256::from(200),
            "Slot B should survive merge"
        );
    }

    #[test]
    fn test_read_after_write_returns_written_value() {
        let mut state = make_state();
        let slot = U256::from(42);
        let value = U256::from(99999);

        write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot, value);

        // Read should return the written value from cache.
        let read_val = read_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot);
        assert_eq!(read_val, value, "Read should return written value");
    }

    /// Simulates the real block execution flow:
    /// 1. StartBlock internal tx writes slot A (baseFee) via write_storage_at
    /// 2. EVM commit for internal tx (empty state)
    /// 3. EVM commit for user tx (modifies different accounts)
    /// 4. Post-commit hook writes slot B (gasBacklog) via write_storage_at
    /// 5. Merge transitions
    /// Both slots should survive in the bundle.
    #[test]
    fn test_write_survives_evm_commit_flow() {
        let mut state = make_state();
        let slot_basefee = U256::from(10);
        let slot_backlog = U256::from(20);

        // Step 1: StartBlock writes baseFee.
        write_storage_at(
            &mut state,
            ARBOS_STATE_ADDRESS,
            slot_basefee,
            U256::from(100_000_000),
        );

        // Step 2: EVM commit for internal tx (empty state).
        use revm_database::DatabaseCommit;
        let empty_state: alloy_primitives::map::HashMap<Address, revm_state::Account> =
            Default::default();
        state.commit(empty_state);

        // Step 3: EVM commit for user tx (modifies a different account).
        let sender = address!("1111111111111111111111111111111111111111");
        let mut user_changes: alloy_primitives::map::HashMap<Address, revm_state::Account> =
            Default::default();
        // Load sender into cache first so commit doesn't panic.
        let _ = state.load_cache_account(sender);
        let mut sender_acct = revm_state::Account::default();
        sender_acct.info.balance = U256::from(1_000_000);
        sender_acct.info.nonce = 1;
        sender_acct.mark_touch();
        user_changes.insert(sender, sender_acct);
        state.commit(user_changes);

        // Step 4: Post-commit hook writes gasBacklog.
        write_storage_at(
            &mut state,
            ARBOS_STATE_ADDRESS,
            slot_backlog,
            U256::from(540_000),
        );

        // Step 5: Merge transitions.
        state.merge_transitions(BundleRetention::Reverts);
        let bundle = state.take_bundle();

        // Both slots should be in the bundle.
        let acct = bundle
            .state
            .get(&ARBOS_STATE_ADDRESS)
            .expect("ArbOS account should be in bundle");
        assert_eq!(
            acct.storage.get(&slot_basefee).unwrap().present_value,
            U256::from(100_000_000),
            "baseFee slot should survive"
        );
        assert_eq!(
            acct.storage.get(&slot_backlog).unwrap().present_value,
            U256::from(540_000),
            "gasBacklog slot should survive"
        );
    }
}
