use std::collections::HashMap;

use alloy_primitives::Address;
use revm::{database::State, Database};
use revm_database::{AccountStatus as CacheAccountStatus, TransitionAccount};
use revm_state::AccountInfo;

#[derive(Clone, Debug)]
struct Entry {
    previous_info: Option<AccountInfo>,
    previous_status: CacheAccountStatus,
}

/// Records pre-mutation snapshots so direct State-cache writes performed
/// outside revm's normal transition flow can be committed back as
/// transitions at end of tx.
///
/// Every captured account — freshly created or modified — is emitted as a
/// manually-built [`TransitionAccount`] whose `previous_info` is the snapshot
/// taken before the first write. This keeps the revert baseline equal to the
/// parent-block state even though `apply_balance_op` pre-mutates the cache for
/// within-tx visibility; a transient pre-write (e.g. a retry tx's prepaid-gas
/// mint) must never become the account's changeset baseline.
#[derive(Default, Debug)]
pub struct StateOverlay {
    entries: HashMap<Address, Entry>,
}

impl StateOverlay {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn reset_tx(&mut self) {
        self.entries.clear();
    }

    pub fn record_pre_touch<DB: Database>(&mut self, state: &mut State<DB>, addr: Address) {
        if self.entries.contains_key(&addr) {
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
        self.entries.insert(
            addr,
            Entry {
                previous_info,
                previous_status,
            },
        );
    }

    pub fn drain_and_apply<DB: Database>(
        &mut self,
        state: &mut State<DB>,
        zombies: &rustc_hash::FxHashSet<Address>,
    ) {
        if self.entries.is_empty() {
            return;
        }
        let entries: Vec<(Address, Entry)> = self.entries.drain().collect();

        let mut existing_transitions: Vec<(Address, TransitionAccount)> = Vec::new();

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
            if pre_empty && cur_empty {
                // A present-empty result is normally pruned (EIP-161). An account
                // resurrected this block by a zero-value transfer on pre-Stylus
                // ArbOS must instead persist as a present-empty leaf. Its revert
                // baseline is the genuinely absent parent state: a snapshot taken
                // from a destructed cache entry can carry a stale non-empty
                // status, so it is normalised to LoadedNotExisting, which reverts
                // to "absent" rather than to a spurious present-empty account.
                if zombies.contains(&addr) && current_info.is_some() {
                    let previous_status = if entry.previous_info.is_none() {
                        CacheAccountStatus::LoadedNotExisting
                    } else {
                        entry.previous_status
                    };
                    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
                        cached.status = CacheAccountStatus::InMemoryChange;
                    }
                    existing_transitions.push((
                        addr,
                        TransitionAccount {
                            info: current_info.clone(),
                            status: CacheAccountStatus::InMemoryChange,
                            previous_info: entry.previous_info,
                            previous_status,
                            storage: Default::default(),
                            storage_was_destroyed: false,
                        },
                    ));
                }
                continue;
            }

            // The emitted transition must be a valid successor of the status
            // the bundle already holds for this account. Within a multi-block
            // batch the cache can be evicted while the bundle keeps an account
            // as `Changed`/`InMemoryChange`; the cache snapshot alone is not
            // authoritative, so prefer the bundle status when present.
            let base_status = state
                .bundle_state
                .state
                .get(&addr)
                .map(|b| b.status)
                .unwrap_or(entry.previous_status);
            let live_in_bundle = state
                .bundle_state
                .state
                .get(&addr)
                .is_some_and(|b| b.info.is_some());

            let was_non_existing = !live_in_bundle
                && (entry.previous_info.is_none()
                    || matches!(
                        base_status,
                        CacheAccountStatus::LoadedNotExisting
                            | CacheAccountStatus::LoadedEmptyEIP161
                    ));

            if was_non_existing && !cur_empty {
                // Insert the new account via an explicit transition whose
                // baseline is the recorded pre-existing state (absent for a
                // freshly-seen account). Committing through revm's create path
                // would capture the transient pre-mutation — e.g. a retry tx's
                // prepaid-gas mint written only for within-tx visibility — as
                // the revert baseline, corrupting the account changeset and the
                // incremental-merkle trie.
                if let Some(cached) = state.cache.accounts.get_mut(&addr) {
                    cached.status = CacheAccountStatus::InMemoryChange;
                }
                existing_transitions.push((
                    addr,
                    TransitionAccount {
                        info: current_info.clone(),
                        status: CacheAccountStatus::InMemoryChange,
                        previous_info: entry.previous_info,
                        previous_status: entry.previous_status,
                        storage: Default::default(),
                        storage_was_destroyed: false,
                    },
                ));
                continue;
            }

            // Non-empty result keeps the account live: map the base status to
            // its modified successor. Empty result deletes it (EIP-161).
            let new_status = if cur_empty {
                match base_status {
                    CacheAccountStatus::LoadedNotExisting => continue,
                    CacheAccountStatus::DestroyedAgain | CacheAccountStatus::DestroyedChanged => {
                        CacheAccountStatus::DestroyedAgain
                    }
                    _ => CacheAccountStatus::Destroyed,
                }
            } else {
                match base_status {
                    CacheAccountStatus::Loaded => CacheAccountStatus::Changed,
                    CacheAccountStatus::LoadedNotExisting
                    | CacheAccountStatus::LoadedEmptyEIP161 => CacheAccountStatus::InMemoryChange,
                    CacheAccountStatus::DestroyedAgain
                    | CacheAccountStatus::Destroyed
                    | CacheAccountStatus::DestroyedChanged => CacheAccountStatus::DestroyedChanged,
                    other => other,
                }
            };

            let goes_destroyed = matches!(
                new_status,
                CacheAccountStatus::Destroyed | CacheAccountStatus::DestroyedAgain
            );
            let transition_info = if goes_destroyed {
                None
            } else {
                current_info.clone()
            };
            let storage_was_destroyed = goes_destroyed && !pre_empty;

            if let Some(cached) = state.cache.accounts.get_mut(&addr) {
                cached.status = new_status;
                if goes_destroyed {
                    cached.account = None;
                }
            }

            existing_transitions.push((
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

        if !existing_transitions.is_empty() {
            state.apply_transition(existing_transitions);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{address, U256};
    use revm::database::{EmptyDB, State};
    use revm_database::states::{
        bundle_state::BundleRetention, cache_account::CacheAccount, plain_account::PlainAccount,
    };

    fn make_state() -> State<EmptyDB> {
        State::builder()
            .with_database(EmptyDB::default())
            .with_bundle_update()
            .build()
    }

    #[test]
    fn fresh_account_credit_lands_in_persisted_bundle() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let recipient = address!("000000000000000000000000000000000000beef");
        let credit = U256::from(0x6a94d74f430000u64);

        overlay.record_pre_touch(&mut state, recipient);
        let entry = state.cache.accounts.get_mut(&recipient).unwrap();
        entry.account = Some(PlainAccount {
            info: AccountInfo {
                balance: credit,
                ..Default::default()
            },
            storage: Default::default(),
        });

        overlay.drain_and_apply(&mut state, &Default::default());
        state.merge_transitions(BundleRetention::Reverts);

        let bundled = state
            .bundle_state
            .state
            .get(&recipient)
            .and_then(|a| a.info.as_ref())
            .map(|i| i.balance);
        assert_eq!(bundled, Some(credit));
    }

    #[test]
    fn existing_account_change_lands_in_persisted_bundle() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let acct = address!("000000000000000000000000000000000000c0de");

        state.cache.accounts.insert(
            acct,
            CacheAccount {
                account: Some(PlainAccount {
                    info: AccountInfo {
                        balance: U256::from(10u64),
                        ..Default::default()
                    },
                    storage: Default::default(),
                }),
                status: CacheAccountStatus::Loaded,
            },
        );

        overlay.record_pre_touch(&mut state, acct);
        if let Some(p) = state
            .cache
            .accounts
            .get_mut(&acct)
            .and_then(|c| c.account.as_mut())
        {
            p.info.balance = U256::from(42u64);
        }

        overlay.drain_and_apply(&mut state, &Default::default());
        state.merge_transitions(BundleRetention::Reverts);

        let bundled = state
            .bundle_state
            .state
            .get(&acct)
            .and_then(|a| a.info.as_ref())
            .map(|i| i.balance);
        assert_eq!(bundled, Some(U256::from(42u64)));
    }

    fn credit<DB: Database>(
        state: &mut State<DB>,
        overlay: &mut StateOverlay,
        a: Address,
        by: U256,
    ) {
        overlay.record_pre_touch(state, a);
        let c = state.cache.accounts.get_mut(&a).unwrap();
        match c.account.as_mut() {
            Some(acct) => acct.info.balance += by,
            None => {
                c.account = Some(PlainAccount {
                    info: AccountInfo {
                        balance: by,
                        ..Default::default()
                    },
                    storage: Default::default(),
                })
            }
        }
        overlay.drain_and_apply(state, &Default::default());
        state.merge_transitions(BundleRetention::Reverts);
        overlay.reset_tx();
    }

    #[test]
    fn repro_fresh_credit_then_recredit_across_merges() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let a = address!("00000000000000000000000000000000feed0001");
        credit(&mut state, &mut overlay, a, U256::from(1_000_000u64));
        credit(&mut state, &mut overlay, a, U256::from(2_000_000u64));
        let bal = state
            .bundle_state
            .state
            .get(&a)
            .and_then(|x| x.info.as_ref())
            .map(|i| i.balance);
        assert_eq!(bal, Some(U256::from(3_000_000u64)));
    }

    /// An account already established in the accumulated bundle (status
    /// `Changed`) is re-credited after its cache entry has been evicted. The
    /// overlay must emit a transition that is a valid successor of the bundle
    /// status, not re-create it as `InMemoryChange` (which revm rejects).
    #[test]
    fn recredit_with_drifted_cache_status_stays_changed() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let a = address!("00000000000000000000000000000000feed0002");

        // Pre-existing on-disk (Loaded) account credited once → bundle `Changed`.
        state.cache.accounts.insert(
            a,
            CacheAccount {
                account: Some(PlainAccount {
                    info: AccountInfo {
                        balance: U256::from(10u64),
                        ..Default::default()
                    },
                    storage: Default::default(),
                }),
                status: CacheAccountStatus::Loaded,
            },
        );
        credit(&mut state, &mut overlay, a, U256::from(1_000_000u64));
        assert_eq!(
            state.bundle_state.state.get(&a).map(|b| b.status),
            Some(CacheAccountStatus::Changed)
        );

        // The cache status drifts to an "empty/non-existing" marker (e.g. via
        // EIP-161 touch handling) while the account and the bundle keep it as a
        // live `Changed` entry. Pre-fix this drove the overlay to emit an
        // `InMemoryChange`/created transition, which revm rejects.
        state.cache.accounts.get_mut(&a).unwrap().status = CacheAccountStatus::LoadedEmptyEIP161;

        credit(&mut state, &mut overlay, a, U256::from(2_000_000u64));
        let acct = state.bundle_state.state.get(&a).unwrap();
        assert_eq!(acct.status, CacheAccountStatus::Changed);
        assert_eq!(
            acct.info.as_ref().map(|i| i.balance),
            Some(U256::from(3_000_010u64))
        );
    }

    #[test]
    fn repro_loaded_credit_then_recredit_across_merges() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let a = address!("00000000000000000000000000000000feed0003");
        // Pre-existing on-disk account (Loaded).
        state.cache.accounts.insert(
            a,
            CacheAccount {
                account: Some(PlainAccount {
                    info: AccountInfo {
                        balance: U256::from(10u64),
                        ..Default::default()
                    },
                    storage: Default::default(),
                }),
                status: CacheAccountStatus::Loaded,
            },
        );
        credit(&mut state, &mut overlay, a, U256::from(1_000_000u64));
        credit(&mut state, &mut overlay, a, U256::from(1_000_000u64));
        let bal = state
            .bundle_state
            .state
            .get(&a)
            .and_then(|x| x.info.as_ref())
            .map(|i| i.balance);
        assert_eq!(bal, Some(U256::from(2_000_010u64)));
    }

    #[test]
    fn created_then_emptied_in_same_tx_produces_no_bundle_entry() {
        let mut state = make_state();
        let mut overlay = StateOverlay::new();
        let transient = address!("0000000000000000000000000000000000007a17");

        overlay.record_pre_touch(&mut state, transient);
        let entry = state.cache.accounts.get_mut(&transient).unwrap();
        entry.account = Some(PlainAccount {
            info: AccountInfo {
                balance: U256::ZERO,
                ..Default::default()
            },
            storage: Default::default(),
        });

        overlay.drain_and_apply(&mut state, &Default::default());
        state.merge_transitions(BundleRetention::Reverts);

        assert!(!state.bundle_state.state.contains_key(&transient));
    }

    #[test]
    fn zombie_resurrection_reverts_to_absent() {
        // A destructed account (cache account=None) carrying a stale status is
        // resurrected present-empty via the zombie path. Forward it must be a
        // present-empty leaf; on a block unwind it must revert to absent rather
        // than be resurrected, whatever stale status the destruct left behind.
        let addr = address!("00000000000000000000000000000000deadbeef");
        for stale in [
            CacheAccountStatus::InMemoryChange,
            CacheAccountStatus::Loaded,
            CacheAccountStatus::LoadedNotExisting,
            CacheAccountStatus::LoadedEmptyEIP161,
        ] {
            let mut state = make_state();
            let mut overlay = StateOverlay::new();
            state.cache.accounts.insert(
                addr,
                CacheAccount {
                    account: None,
                    status: stale,
                },
            );
            overlay.record_pre_touch(&mut state, addr);
            let entry = state.cache.accounts.get_mut(&addr).unwrap();
            entry.account = Some(PlainAccount {
                info: AccountInfo::default(),
                storage: Default::default(),
            });
            entry.status = CacheAccountStatus::InMemoryChange;

            let zombies: rustc_hash::FxHashSet<Address> = std::iter::once(addr).collect();
            overlay.drain_and_apply(&mut state, &zombies);
            state.merge_transitions(BundleRetention::Reverts);

            let forward = state
                .bundle_state
                .state
                .get(&addr)
                .and_then(|a| a.info.as_ref())
                .cloned();
            assert!(
                forward.as_ref().is_some_and(|i| i.is_empty()),
                "stale={stale:?}: zombie must persist present-empty forward, got {forward:?}"
            );

            state.bundle_state.revert(usize::MAX);
            let reverted = state
                .bundle_state
                .state
                .get(&addr)
                .and_then(|a| a.info.as_ref())
                .cloned();
            assert!(
                reverted.is_none(),
                "stale={stale:?}: unwound zombie must be absent, got {reverted:?}"
            );
        }
    }
}
