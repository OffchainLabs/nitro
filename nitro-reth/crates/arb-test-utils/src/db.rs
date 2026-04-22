//! In-memory `Database` impl for tests.
//!
//! Returns zero/default for every read so the cache layer above can serve
//! everything that gets written during a test.

use alloy_primitives::{keccak256, Address, B256, U256};
use revm::{
    database::{states::account_status::AccountStatus, PlainAccount, State},
    state::{AccountInfo, Bytecode},
    Database,
};

/// Database that returns empty/zero for all reads.
#[derive(Debug, Default, Clone, Copy)]
pub struct EmptyDb;

impl Database for EmptyDb {
    type Error = std::convert::Infallible;

    fn basic(&mut self, _address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        Ok(None)
    }

    fn code_by_hash(&mut self, _code_hash: B256) -> Result<Bytecode, Self::Error> {
        Ok(Bytecode::default())
    }

    fn storage(&mut self, _address: Address, _index: U256) -> Result<U256, Self::Error> {
        Ok(U256::ZERO)
    }

    fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
        Ok(B256::ZERO)
    }
}

/// Ensure `addr` exists in the cache as a fresh account. Required before
/// the first storage write so revm tracks it for bundle merging.
pub fn ensure_cache_account(state: &mut State<EmptyDb>, addr: Address) {
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        if cached.account.is_none() {
            cached.account = Some(PlainAccount {
                info: AccountInfo {
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
