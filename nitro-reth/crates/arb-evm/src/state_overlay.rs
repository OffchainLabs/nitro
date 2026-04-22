//! Snapshots `(info, status)` before each helper cache mutation and at
//! end of tx pushes one explicit `TransitionAccount` per real change
//! through `State::apply_transition`.

use std::{cell::RefCell, collections::HashMap};

use alloy_consensus::constants::KECCAK_EMPTY;
use alloy_primitives::Address;
use revm::{database::State, Database};
use revm_database::{AccountStatus as CacheAccountStatus, TransitionAccount};
use revm_state::AccountInfo;

#[derive(Clone, Debug)]
struct Entry {
    previous_info: Option<AccountInfo>,
    previous_status: CacheAccountStatus,
}

thread_local! {
    static OVERLAY: RefCell<HashMap<Address, Entry>> = RefCell::new(HashMap::new());
}

pub fn reset_tx() {
    OVERLAY.with(|o| o.borrow_mut().clear());
}

/// Snapshot pre-mutation `(info, status)` for `addr`; idempotent per tx.
pub fn record_pre_touch<DB: Database>(state: &mut State<DB>, addr: Address) {
    let already = OVERLAY.with(|o| o.borrow().contains_key(&addr));
    if already {
        return;
    }
    let _ = state.load_cache_account(addr);
    let cache_entry = state.cache.accounts.get(&addr);
    let previous_info = cache_entry
        .and_then(|c| c.account.as_ref())
        .map(|a| a.info.clone());
    let previous_status = cache_entry
        .map(|c| c.status)
        .unwrap_or(CacheAccountStatus::LoadedNotExisting);
    OVERLAY.with(|o| {
        o.borrow_mut().insert(
            addr,
            Entry {
                previous_info,
                previous_status,
            },
        )
    });
}

/// Push one `TransitionAccount` per address whose info actually changed.
pub fn drain_and_apply<DB: Database>(state: &mut State<DB>) {
    let entries: Vec<(Address, Entry)> = OVERLAY.with(|o| o.borrow_mut().drain().collect());
    if entries.is_empty() {
        return;
    }

    let mut transitions = Vec::with_capacity(entries.len());
    for (addr, entry) in entries {
        let current_info = state
            .cache
            .accounts
            .get(&addr)
            .and_then(|c| c.account.as_ref())
            .map(|a| a.info.clone());

        if current_info == entry.previous_info {
            continue;
        }

        let pre_empty = entry
            .previous_info
            .as_ref()
            .map(|i| i.is_empty())
            .unwrap_or(true);
        let cur_empty = current_info.as_ref().map(|i| i.is_empty()).unwrap_or(true);

        let new_status = match (pre_empty, cur_empty) {
            (true, true) => continue,
            (true, false) => entry.previous_status.on_created(),
            (false, true) => entry.previous_status.on_touched_empty_post_eip161(),
            (false, false) => {
                let prev = entry.previous_info.as_ref();
                let had_no_nonce_and_code = prev
                    .map(|i| i.nonce == 0 && i.code_hash == KECCAK_EMPTY)
                    .unwrap_or(true);
                entry.previous_status.on_changed(had_no_nonce_and_code)
            }
        };

        let goes_destroyed = matches!(
            new_status,
            CacheAccountStatus::Destroyed | CacheAccountStatus::DestroyedAgain
        );
        let transition_info = if goes_destroyed { None } else { current_info };
        let storage_was_destroyed = goes_destroyed && !pre_empty;

        if let Some(cached) = state.cache.accounts.get_mut(&addr) {
            cached.status = new_status;
            if goes_destroyed {
                cached.account = None;
            }
        }

        transitions.push((
            addr,
            TransitionAccount {
                info: transition_info,
                status: new_status,
                previous_info: entry.previous_info,
                previous_status: entry.previous_status,
                storage: Default::default(),
                storage_was_destroyed,
            },
        ));
    }

    if !transitions.is_empty() {
        state.apply_transition(transitions);
    }
}
