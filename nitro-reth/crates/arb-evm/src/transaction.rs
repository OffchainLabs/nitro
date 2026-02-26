use alloy_eips::eip2930::AccessList;
use alloy_evm::tx::{FromRecoveredTx, FromTxWithEncoded};
use alloy_primitives::{Address, Bytes, U256};
use reth_ethereum_primitives::TransactionSigned;
use revm::context::TxEnv;

use arb_primitives::tx_types::ArbTxType;

/// Wrapper around revm's TxEnv for Arbitrum transaction types.
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
}

impl alloy_evm::IntoTxEnv<ArbTransaction> for ArbTransaction {
    fn into_tx_env(self) -> ArbTransaction {
        self
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
