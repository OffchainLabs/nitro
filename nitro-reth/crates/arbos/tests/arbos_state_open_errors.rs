use std::cell::Cell;

use alloy_primitives::{Address, B256, U256};
use arb_storage::{DatabaseError, StorageError};
use arbos::{
    arbos_state::{ArbosState, ArbosStateError},
    burn::SystemBurner,
};
use revm::Database;
use revm_database::StateBuilder;
use revm_database_interface::DBErrorMarker;

#[derive(Debug)]
struct FakeDbError(&'static str);

impl std::fmt::Display for FakeDbError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.0)
    }
}

impl std::error::Error for FakeDbError {}
impl DBErrorMarker for FakeDbError {}

/// Database that errors on the very next `storage()` call.
struct FailingDb {
    armed: Cell<bool>,
}

impl FailingDb {
    fn new() -> Self {
        Self {
            armed: Cell::new(false),
        }
    }
}

impl Database for FailingDb {
    type Error = FakeDbError;

    fn basic(&mut self, _address: Address) -> Result<Option<revm_state::AccountInfo>, Self::Error> {
        Ok(None)
    }

    fn code_by_hash(&mut self, _code_hash: B256) -> Result<revm_state::Bytecode, Self::Error> {
        Ok(revm_state::Bytecode::default())
    }

    fn storage(&mut self, _address: Address, _index: U256) -> Result<U256, Self::Error> {
        if self.armed.replace(false) {
            return Err(FakeDbError("synthetic db read failure"));
        }
        Ok(U256::ZERO)
    }

    fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
        Ok(B256::ZERO)
    }
}

#[test]
fn open_propagates_db_storage_failure() {
    let db = FailingDb::new();
    db.armed.set(true);
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    match ArbosState::open(&mut state, SystemBurner::new(None, false)) {
        Err(ArbosStateError::Storage(StorageError::Database(DatabaseError::Read(_)))) => {}
        Err(other) => {
            panic!("expected ArbosStateError::Storage(Database(Read(_))), got {other:?}")
        }
        Ok(_) => panic!("expected error, got Ok"),
    }
}

#[test]
fn open_returns_uninitialised_for_zero_version_slot() {
    let db = FailingDb::new();
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    match ArbosState::open(&mut state, SystemBurner::new(None, false)) {
        Err(ArbosStateError::Uninitialised) => {}
        Err(other) => panic!("expected ArbosStateError::Uninitialised, got {other:?}"),
        Ok(_) => panic!("expected error, got Ok"),
    }
}
