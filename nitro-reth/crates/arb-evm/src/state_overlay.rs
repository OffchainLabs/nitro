//! Per-tx overlay that turns direct `State::cache.accounts` mutations
//! into proper revm transitions so they appear in `ExecutionOutput.state`.

use std::{cell::RefCell, collections::HashMap};

use alloy_primitives::Address;
use revm::{database::State, Database};
use revm_database::states::plain_account::PlainAccount;
use revm_state::{Account, AccountInfo, AccountStatus, EvmState};

#[derive(Clone, Debug)]
struct Entry {
    previous_info: Option<AccountInfo>,
}

thread_local! {
    static OVERLAY: RefCell<HashMap<Address, Entry>> = RefCell::new(HashMap::new());
}

pub fn reset_tx() {
    OVERLAY.with(|o| o.borrow_mut().clear());
}

/// Snapshot pre-tx `AccountInfo` for `addr` once per tx. Call before mutating cache.
pub fn record_pre_touch<DB: Database>(state: &mut State<DB>, addr: Address) {
    let already = OVERLAY.with(|o| o.borrow().contains_key(&addr));
    if already {
        return;
    }
    let _ = state.load_cache_account(addr);
    let previous_info = state
        .cache
        .accounts
        .get(&addr)
        .and_then(|c| c.account.as_ref())
        .map(|a| a.info.clone());
    OVERLAY.with(|o| o.borrow_mut().insert(addr, Entry { previous_info }));
}

/// Restore each touched cache entry to its pre-tx info and return an
/// `EvmState` carrying the post-mutation values for revm to commit.
pub fn drain_and_restore<DB: Database>(state: &mut State<DB>) -> EvmState {
    let entries: Vec<(Address, Entry)> = OVERLAY.with(|o| o.borrow_mut().drain().collect());

    let mut evm_state = EvmState::default();
    for (addr, entry) in entries {
        let current = state
            .cache
            .accounts
            .get(&addr)
            .and_then(|c| c.account.as_ref())
            .map(|a| a.info.clone());

        let cached = match state.cache.accounts.get_mut(&addr) {
            Some(c) => c,
            None => continue,
        };
        match &entry.previous_info {
            Some(info) => match cached.account {
                Some(ref mut acct) => acct.info = info.clone(),
                None => {
                    cached.account = Some(PlainAccount {
                        info: info.clone(),
                        storage: Default::default(),
                    });
                }
            },
            None => {
                cached.account = None;
            }
        }

        let info = current.unwrap_or_default();
        let mut account = Account::from(info.clone());
        account.status |= AccountStatus::Touched;
        if entry.previous_info.is_none() && !info.is_empty() {
            account.status |= AccountStatus::Created;
        }
        evm_state.insert(addr, account);
    }
    evm_state
}
