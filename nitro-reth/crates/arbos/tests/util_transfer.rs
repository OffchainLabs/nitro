use std::cell::RefCell;

use alloy_primitives::{address, Address, U256};
use arbos::util::{burn_balance, mint_balance, transfer_balance};

type TransferLog = RefCell<Vec<(Option<Address>, Option<Address>, U256)>>;

fn record(log: &TransferLog) -> impl FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), ()> + '_ {
    move |from, to, amount| {
        log.borrow_mut().push((from.copied(), to.copied(), amount));
        Ok(())
    }
}

#[test]
fn transfer_records_from_and_to() {
    let log = TransferLog::default();
    let a = address!("AAAA000000000000000000000000000000000000");
    let b = address!("BBBB000000000000000000000000000000000000");
    transfer_balance(Some(&a), Some(&b), U256::from(42u64), record(&log)).unwrap();
    assert_eq!(*log.borrow(), vec![(Some(a), Some(b), U256::from(42u64))]);
}

#[test]
fn mint_passes_none_from() {
    let log = TransferLog::default();
    let to = address!("AAAA000000000000000000000000000000000000");
    mint_balance(&to, U256::from(100u64), record(&log)).unwrap();
    assert_eq!(*log.borrow(), vec![(None, Some(to), U256::from(100u64))]);
}

#[test]
fn burn_passes_none_to() {
    let log = TransferLog::default();
    let from = address!("BBBB000000000000000000000000000000000000");
    burn_balance(&from, U256::from(50u64), record(&log)).unwrap();
    assert_eq!(*log.borrow(), vec![(Some(from), None, U256::from(50u64))]);
}

#[test]
fn transfer_propagates_state_fn_error() {
    let result = transfer_balance(
        Some(&address!("AAAA000000000000000000000000000000000000")),
        Some(&address!("BBBB000000000000000000000000000000000000")),
        U256::from(1u64),
        |_, _, _| Err(()),
    );
    assert!(result.is_err());
}
