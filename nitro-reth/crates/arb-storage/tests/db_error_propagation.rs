//! Regression coverage for the live consensus bug fix: ensure a database read
//! failure surfaces as `StorageError::Database` instead of being silently
//! converted to a zero value.

use std::{cell::Cell, convert::Infallible};

use alloy_primitives::{Address, B256, U256};
use arb_storage::{
    read_arbos_storage, read_storage_at, write_arbos_storage, write_storage_at, DatabaseError,
    StorageError, ARBOS_STATE_ADDRESS,
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

/// Database that returns an error on the very next `storage()` call.
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
fn read_storage_at_propagates_database_error() {
    let db = FailingDb::new();
    db.armed.set(true);
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    let err = read_storage_at(&mut state, ARBOS_STATE_ADDRESS, U256::from(42))
        .expect_err("db read returned Err; the call should propagate");
    match err {
        StorageError::Database(DatabaseError::Read(_)) => {}
        other => panic!("expected StorageError::Database(Read(_)), got {other:?}"),
    }
}

#[test]
fn read_arbos_storage_propagates_database_error() {
    let db = FailingDb::new();
    db.armed.set(true);
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    let err = read_arbos_storage(&mut state, U256::from(7))
        .expect_err("db read returned Err; the call should propagate");
    assert!(matches!(
        err,
        StorageError::Database(DatabaseError::Read(_))
    ));
}

#[test]
fn write_storage_at_propagates_db_lookup_error() {
    let db = FailingDb::new();
    db.armed.set(true);
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    let err = write_storage_at(
        &mut state,
        ARBOS_STATE_ADDRESS,
        U256::from(13),
        U256::from(99),
    )
    .expect_err("the original-value lookup goes through database.storage()");
    assert!(matches!(
        err,
        StorageError::Database(DatabaseError::Read(_))
    ));
}

#[test]
fn write_arbos_storage_propagates_db_lookup_error() {
    let db = FailingDb::new();
    db.armed.set(true);
    let mut state = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

    let err = write_arbos_storage(&mut state, U256::from(13), U256::from(99))
        .expect_err("the original-value lookup goes through database.storage()");
    assert!(matches!(
        err,
        StorageError::Database(DatabaseError::Read(_))
    ));
}

/// Cache-first reads must not consult the database, so a primed DB failure
/// goes unobserved when the value is already cached.
#[test]
fn cache_hit_short_circuits_database_failure() {
    #[derive(Default)]
    struct InfallibleDb;

    impl Database for InfallibleDb {
        type Error = Infallible;
        fn basic(
            &mut self,
            _address: Address,
        ) -> Result<Option<revm_state::AccountInfo>, Self::Error> {
            Ok(None)
        }
        fn code_by_hash(&mut self, _code_hash: B256) -> Result<revm_state::Bytecode, Self::Error> {
            Ok(revm_state::Bytecode::default())
        }
        fn storage(&mut self, _address: Address, _index: U256) -> Result<U256, Self::Error> {
            Ok(U256::ZERO)
        }
        fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
            Ok(B256::ZERO)
        }
    }

    let mut state = StateBuilder::new()
        .with_database(InfallibleDb)
        .with_bundle_update()
        .build();

    let slot = U256::from(1);
    let value = U256::from(0x1234);
    write_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot, value)
        .expect("write must succeed with infallible db");
    let got = read_storage_at(&mut state, ARBOS_STATE_ADDRESS, slot)
        .expect("read must succeed via cache");
    assert_eq!(got, value);
}
