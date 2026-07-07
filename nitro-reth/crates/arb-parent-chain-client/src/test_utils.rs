//! Builders for constructing alloy values in tests, shared across the crate's
//! test modules.

use alloy_consensus::{
    transaction::{Recovered, TransactionInfo},
    ReceiptEnvelope, SignableTransaction, TxEip1559, TxEnvelope,
};
use alloy_primitives::{Address, Signature, B256, U256};
use alloy_rpc_types_eth::{Block, BlockTransactions, Header, Log, Transaction, TransactionReceipt};

/// Builds a header with the given number and hash.
pub(crate) fn header(number: u64, hash: B256) -> Header {
    let mut h: Header = Header::default();
    h.inner.number = number;
    h.hash = hash;
    h
}

/// Builds a log emitted at `block` by `address`.
pub(crate) fn log_at(block: u64, address: Address) -> Log {
    let mut log = Log::default();
    log.inner.address = address;
    log.block_number = Some(block);
    log.block_hash = Some(B256::repeat_byte(block as u8));
    log
}

/// Builds a transaction. `nonce` gives it a distinct hash; `mined_at` sets its
/// `(block hash, index)` (leave `None` for a pending transaction).
pub(crate) fn tx(nonce: u64, mined_at: Option<(B256, u64)>) -> Transaction {
    let signed = TxEip1559 {
        nonce,
        ..Default::default()
    }
    .into_signed(Signature::new(U256::ZERO, U256::ZERO, false));
    let recovered = Recovered::new_unchecked(TxEnvelope::Eip1559(signed), Address::ZERO);
    let info = TransactionInfo {
        hash: None,
        index: mined_at.map(|(_, index)| index),
        block_hash: mined_at.map(|(hash, _)| hash),
        block_number: mined_at.map(|_| 0),
        base_fee: None,
    };
    Transaction::from_transaction(recovered, info)
}

/// Builds a block with the given number, hash, and transactions.
pub(crate) fn block(number: u64, hash: B256, txs: BlockTransactions<Transaction>) -> Block {
    Block::new(header(number, hash), txs)
}

/// Builds a receipt for the given transaction hash.
pub(crate) fn receipt(tx_hash: B256) -> TransactionReceipt {
    TransactionReceipt {
        inner: ReceiptEnvelope::Eip1559(Default::default()),
        transaction_hash: tx_hash,
        transaction_index: None,
        block_hash: None,
        block_number: None,
        gas_used: 0,
        effective_gas_price: 0,
        blob_gas_used: None,
        blob_gas_price: None,
        from: Address::ZERO,
        to: None,
        contract_address: None,
    }
}
