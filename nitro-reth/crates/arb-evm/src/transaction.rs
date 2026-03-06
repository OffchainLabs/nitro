use alloy_consensus::Transaction;
use alloy_eips::eip2930::AccessList;
use alloy_evm::tx::{FromRecoveredTx, FromTxWithEncoded, IntoTxEnv};
use alloy_primitives::{Address, Bytes, U256};
use arb_primitives::ArbTransactionSigned;
use reth_ethereum_primitives::TransactionSigned;
use revm::context::TxEnv;

use arb_primitives::tx_types::ArbTxType;

/// Helper for building Arbitrum-specific TxEnv values.
///
/// Handles Arbitrum-specific conversion rules:
/// - Internal/Deposit txs get 1M gas if zero, gas_price=0
/// - SubmitRetryable txs use gas_price=0 (no coinbase tips)
/// - Retry txs preserve value for ETH transfers
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct ArbTransaction(pub TxEnv);

impl ArbTransaction {
    /// Create an ArbTransaction from the raw components of an Arbitrum tx.
    pub fn from_parts(
        sender: Address,
        tx_type: ArbTxType,
        gas_limit: u64,
        gas_price: u128,
        value: U256,
        to: revm::primitives::TxKind,
        data: alloy_primitives::Bytes,
        nonce: u64,
        chain_id: Option<u64>,
    ) -> Self {
        let mut tx = TxEnv::default();
        tx.caller = sender;
        tx.gas_limit = gas_limit;

        // Internal/Deposit txs get minimum 1M gas
        if matches!(tx_type, ArbTxType::ArbitrumInternalTx | ArbTxType::ArbitrumDepositTx)
            && gas_limit == 0
        {
            tx.gas_limit = 1_000_000;
        }

        tx.gas_priority_fee = Some(0);

        match tx_type {
            ArbTxType::ArbitrumDepositTx | ArbTxType::ArbitrumInternalTx => {
                tx.value = U256::ZERO;
                tx.gas_price = 0;
            }
            ArbTxType::ArbitrumSubmitRetryableTx => {
                tx.value = U256::ZERO;
                tx.gas_price = 0;
            }
            _ => {
                tx.value = value;
                tx.gas_price = gas_price;
            }
        }

        tx.kind = to;
        tx.data = data;
        tx.nonce = nonce;
        tx.chain_id = chain_id;

        ArbTransaction(tx)
    }

    /// Unwrap into the inner TxEnv.
    pub fn into_inner(self) -> TxEnv {
        self.0
    }
}

impl From<ArbTransaction> for TxEnv {
    fn from(arb_tx: ArbTransaction) -> Self {
        arb_tx.0
    }
}

impl IntoTxEnv<ArbTransaction> for ArbTransaction {
    fn into_tx_env(self) -> ArbTransaction {
        self
    }
}

impl revm::context_interface::Transaction for ArbTransaction {
    type AccessListItem<'a> = <TxEnv as revm::context_interface::Transaction>::AccessListItem<'a>
        where Self: 'a;
    type Authorization<'a> = <TxEnv as revm::context_interface::Transaction>::Authorization<'a>
        where Self: 'a;

    fn tx_type(&self) -> u8 {
        revm::context_interface::Transaction::tx_type(&self.0)
    }
    fn caller(&self) -> Address {
        revm::context_interface::Transaction::caller(&self.0)
    }
    fn gas_limit(&self) -> u64 {
        revm::context_interface::Transaction::gas_limit(&self.0)
    }
    fn value(&self) -> U256 {
        revm::context_interface::Transaction::value(&self.0)
    }
    fn input(&self) -> &alloy_primitives::Bytes {
        revm::context_interface::Transaction::input(&self.0)
    }
    fn nonce(&self) -> u64 {
        revm::context_interface::Transaction::nonce(&self.0)
    }
    fn kind(&self) -> alloy_primitives::TxKind {
        revm::context_interface::Transaction::kind(&self.0)
    }
    fn chain_id(&self) -> Option<u64> {
        revm::context_interface::Transaction::chain_id(&self.0)
    }
    fn gas_price(&self) -> u128 {
        revm::context_interface::Transaction::gas_price(&self.0)
    }
    fn access_list(&self) -> Option<impl Iterator<Item = Self::AccessListItem<'_>>> {
        revm::context_interface::Transaction::access_list(&self.0)
    }
    fn blob_versioned_hashes(&self) -> &[alloy_primitives::B256] {
        revm::context_interface::Transaction::blob_versioned_hashes(&self.0)
    }
    fn max_fee_per_blob_gas(&self) -> u128 {
        revm::context_interface::Transaction::max_fee_per_blob_gas(&self.0)
    }
    fn authorization_list_len(&self) -> usize {
        revm::context_interface::Transaction::authorization_list_len(&self.0)
    }
    fn authorization_list(&self) -> impl Iterator<Item = Self::Authorization<'_>> {
        revm::context_interface::Transaction::authorization_list(&self.0)
    }
    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        revm::context_interface::Transaction::max_priority_fee_per_gas(&self.0)
    }
}

impl reth_evm::TransactionEnv for ArbTransaction {
    fn set_gas_limit(&mut self, gas_limit: u64) {
        self.0.gas_limit = gas_limit;
    }

    fn nonce(&self) -> u64 {
        self.0.nonce
    }

    fn set_nonce(&mut self, nonce: u64) {
        self.0.nonce = nonce;
    }

    fn set_access_list(&mut self, access_list: AccessList) {
        self.0.access_list = access_list;
    }
}

impl crate::build::ArbTransactionEnv for ArbTransaction {
    fn set_gas_price(&mut self, gas_price: u128) {
        self.0.gas_price = gas_price;
    }
    fn set_gas_priority_fee(&mut self, fee: Option<u128>) {
        self.0.gas_priority_fee = fee;
    }
}

impl FromRecoveredTx<TransactionSigned> for ArbTransaction {
    fn from_recovered_tx(tx: &TransactionSigned, sender: Address) -> Self {
        ArbTransaction(TxEnv::from_recovered_tx(tx, sender))
    }
}

impl FromTxWithEncoded<TransactionSigned> for ArbTransaction {
    fn from_encoded_tx(tx: &TransactionSigned, sender: Address, encoded: Bytes) -> Self {
        ArbTransaction(TxEnv::from_encoded_tx(tx, sender, encoded))
    }
}

/// Convert an ArbTransactionSigned into a TxEnv for EVM execution.
fn arb_tx_to_tx_env(tx: &ArbTransactionSigned, sender: Address) -> TxEnv {
    use alloy_consensus::Typed2718;
    let arb_type = ArbTxType::from_u8(Typed2718::ty(tx)).ok();
    let is_system_tx = matches!(
        arb_type,
        Some(ArbTxType::ArbitrumInternalTx | ArbTxType::ArbitrumDepositTx)
    );
    let is_submit_retryable = arb_type == Some(ArbTxType::ArbitrumSubmitRetryableTx);

    let mut env = TxEnv::default();
    env.caller = sender;
    env.gas_limit = tx.gas_limit();
    env.nonce = tx.nonce();
    env.chain_id = tx.chain_id();
    env.kind = tx.to().map_or(revm::primitives::TxKind::Create, revm::primitives::TxKind::Call);
    env.data = tx.input().clone();
    env.gas_priority_fee = Some(0);

    if is_system_tx {
        env.gas_price = 0;
        env.value = U256::ZERO;
        if env.gas_limit == 0 {
            env.gas_limit = 1_000_000;
        }
    } else if is_submit_retryable {
        env.gas_price = 0;
        env.value = U256::ZERO;
    } else {
        env.gas_price = tx.max_fee_per_gas();
        env.value = tx.value();
    }

    if let Some(al) = tx.access_list() {
        env.access_list = al.clone();
    }

    env
}

impl FromRecoveredTx<ArbTransactionSigned> for ArbTransaction {
    fn from_recovered_tx(tx: &ArbTransactionSigned, sender: Address) -> Self {
        ArbTransaction(arb_tx_to_tx_env(tx, sender))
    }
}

impl FromTxWithEncoded<ArbTransactionSigned> for ArbTransaction {
    fn from_encoded_tx(tx: &ArbTransactionSigned, sender: Address, _encoded: Bytes) -> Self {
        ArbTransaction(arb_tx_to_tx_env(tx, sender))
    }
}
