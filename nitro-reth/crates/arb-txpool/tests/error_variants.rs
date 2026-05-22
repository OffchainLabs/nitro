use alloy_primitives::{address, U256};
use arb_txpool::{TxPoolError, TxPoolResult};

fn assert_send_sync_static<T: Send + Sync + 'static>() {}

#[test]
fn error_type_is_send_sync_static() {
    assert_send_sync_static::<TxPoolError>();
}

#[test]
fn result_alias_resolves() {
    fn produces_ok() -> TxPoolResult<u32> {
        Ok(7)
    }
    fn produces_err() -> TxPoolResult<u32> {
        Err(TxPoolError::BlobTransactionsDisallowed)
    }
    assert_eq!(produces_ok().unwrap(), 7);
    assert!(matches!(
        produces_err().unwrap_err(),
        TxPoolError::BlobTransactionsDisallowed
    ));
}

#[test]
fn system_transaction_type_carries_type_byte() {
    let err = TxPoolError::SystemTransactionType { type_byte: 0x6E };
    assert!(err.to_string().contains("0x6e"));
    assert!(err.is_bad_transaction());
}

#[test]
fn oversized_data_renders_both_bounds() {
    let err = TxPoolError::OversizedData {
        size: 1_000_000,
        max: 131_072,
    };
    let text = err.to_string();
    assert!(text.contains("1000000"));
    assert!(text.contains("131072"));
    assert!(err.is_bad_transaction());
}

#[test]
fn intrinsic_gas_too_low_is_bad_tx() {
    let err = TxPoolError::IntrinsicGasTooLow {
        provided: 10_000,
        required: 21_000,
    };
    assert!(err.is_bad_transaction());
    assert!(err.to_string().contains("10000"));
    assert!(err.to_string().contains("21000"));
}

#[test]
fn underpriced_is_not_bad_tx() {
    let err = TxPoolError::Underpriced {
        tip: 0,
        base_fee: 1_000_000_000,
    };
    assert!(!err.is_bad_transaction());
}

#[test]
fn nonce_too_low_is_not_bad_tx() {
    let err = TxPoolError::NonceTooLow { tx: 3, sender: 5 };
    assert!(!err.is_bad_transaction());
}

#[test]
fn insufficient_funds_carries_balance_and_cost() {
    let err = TxPoolError::InsufficientFunds {
        required: U256::from(100u64),
        available: U256::from(10u64),
    };
    assert!(!err.is_bad_transaction());
    let text = err.to_string();
    assert!(text.contains("100"));
    assert!(text.contains("10"));
}

#[test]
fn blocked_addresses_categorised_as_bad_tx() {
    let who = address!("00000000000000000000000000000000deadbeef");
    let s = TxPoolError::BlockedSender(who);
    let d = TxPoolError::BlockedDestination(who);
    assert!(s.is_bad_transaction());
    assert!(d.is_bad_transaction());
    assert!(s.to_string().to_lowercase().contains("deadbeef"));
    assert!(d.to_string().to_lowercase().contains("deadbeef"));
}

#[test]
fn chain_id_mismatch_carries_ids() {
    let err = TxPoolError::ChainIdMismatch {
        tx: 1,
        expected: 42_161,
    };
    assert!(err.is_bad_transaction());
    let text = err.to_string();
    assert!(text.contains('1'));
    assert!(text.contains("42161"));
}
