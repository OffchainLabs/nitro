//! Verifies that a transfer-callback shortfall propagates as
//! `RetryableError::Balance(BalanceError::InsufficientBalance { .. })`
//! through the retryable subsystem's typed error surface.

use alloy_primitives::{address, b256, Address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::{
    retryables::{retryable_escrow_address, RetryableError},
    util::BalanceError,
};

const FROM: Address = address!("00000000000000000000000000000000000A11CE");
const BENEFICIARY: Address = address!("00000000000000000000000000000000000B0B00");
const DEST: Address = address!("00000000000000000000000000000000C4A841E0");
const TICKET_ID: B256 = b256!("0000000000000000000000000000000000000000000000000000000000000042");

#[test]
fn delete_propagates_insufficient_balance_as_retryable_error() {
    let mut h = ArbosHarness::new().initialize();

    let state_ptr = h.state_ptr();
    {
        let rs = h.retryable_state();
        let b = unsafe { &mut *state_ptr };
        rs.create_retryable(
            b,
            TICKET_ID,
            1_000,
            FROM,
            Some(DEST),
            U256::from(1_000_000u64),
            BENEFICIARY,
            b"data",
        )
        .unwrap();
    }

    let escrow = retryable_escrow_address(TICKET_ID);
    let requested = U256::from(1_000_000u64);

    let result: Result<bool, RetryableError> = {
        let rs = h.retryable_state();
        rs.delete_retryable(
            unsafe { &mut *state_ptr },
            TICKET_ID,
            |from, _to, amount| {
                Err(BalanceError::InsufficientBalance {
                    account: from,
                    available: U256::ZERO,
                    requested: amount,
                })
            },
            |_addr| requested,
        )
    };

    let err = result.expect_err("delete should propagate the callback error");
    match err {
        RetryableError::Balance(BalanceError::InsufficientBalance {
            account,
            available,
            requested: req,
        }) => {
            assert_eq!(account, escrow);
            assert_eq!(available, U256::ZERO);
            assert_eq!(req, requested);
        }
        other => panic!("unexpected error variant: {other:?}"),
    }
}
