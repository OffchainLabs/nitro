use alloc::vec::Vec;
use core::{
    hash::{Hash, Hasher},
    ops::Deref,
};

use alloy_consensus::{
    transaction::{RlpEcdsaDecodableTx, RlpEcdsaEncodableTx, TxHashRef},
    SignableTransaction, Transaction as ConsensusTx, TxLegacy, Typed2718,
};
use alloy_eips::eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718, IsTyped2718};
use alloy_primitives::{keccak256, Address, Bytes, Signature, TxHash, TxKind, B256, U256};
use alloy_rlp::{Decodable, Encodable};
use reth_primitives_traits::{
    crypto::secp256k1::{recover_signer, recover_signer_unchecked},
    InMemorySize, SignedTransaction,
};

use arb_alloy_consensus::tx::{
    ArbContractTx, ArbDepositTx, ArbInternalTx, ArbRetryTx, ArbSubmitRetryableTx, ArbTxType,
    ArbUnsignedTx,
};

/// Internal ArbOS address used as sender for internal transactions.
const ARBOS_ADDRESS: Address =
    alloy_primitives::address!("00000000000000000000000000000000000A4B05");

/// Retryable precompile address (0x6e).
const RETRYABLE_ADDRESS: Address =
    alloy_primitives::address!("000000000000000000000000000000000000006e");

/// Wraps all supported transaction types (standard Ethereum + Arbitrum-specific).
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum ArbTypedTransaction {
    Deposit(ArbDepositTx),
    Unsigned(ArbUnsignedTx),
    Contract(ArbContractTx),
    Retry(ArbRetryTx),
    SubmitRetryable(ArbSubmitRetryableTx),
    Internal(ArbInternalTx),

    Legacy(TxLegacy),
    Eip2930(alloy_consensus::TxEip2930),
    Eip1559(alloy_consensus::TxEip1559),
    Eip4844(alloy_consensus::TxEip4844),
    Eip7702(alloy_consensus::TxEip7702),
}

/// Discriminant for transaction type classification.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ArbTxTypeLocal {
    Deposit,
    Unsigned,
    Contract,
    Retry,
    SubmitRetryable,
    Internal,
    Legacy,
    Eip2930,
    Eip1559,
    Eip4844,
    Eip7702,
}

impl ArbTxTypeLocal {
    /// Convert to the EIP-2718 type byte.
    pub fn as_u8(self) -> u8 {
        match self {
            Self::Legacy => 0x00,
            Self::Eip2930 => 0x01,
            Self::Eip1559 => 0x02,
            Self::Eip4844 => 0x03,
            Self::Eip7702 => 0x04,
            Self::Deposit => ArbTxType::ArbitrumDepositTx.as_u8(),
            Self::Unsigned => ArbTxType::ArbitrumUnsignedTx.as_u8(),
            Self::Contract => ArbTxType::ArbitrumContractTx.as_u8(),
            Self::Retry => ArbTxType::ArbitrumRetryTx.as_u8(),
            Self::SubmitRetryable => ArbTxType::ArbitrumSubmitRetryableTx.as_u8(),
            Self::Internal => ArbTxType::ArbitrumInternalTx.as_u8(),
        }
    }
}

impl Typed2718 for ArbTxTypeLocal {
    fn is_legacy(&self) -> bool {
        matches!(self, Self::Legacy)
    }

    fn ty(&self) -> u8 {
        self.as_u8()
    }
}

impl alloy_consensus::TransactionEnvelope for ArbTransactionSigned {
    type TxType = ArbTxTypeLocal;

    fn tx_type(&self) -> Self::TxType {
        match &self.transaction {
            ArbTypedTransaction::Legacy(_) => ArbTxTypeLocal::Legacy,
            ArbTypedTransaction::Eip2930(_) => ArbTxTypeLocal::Eip2930,
            ArbTypedTransaction::Eip1559(_) => ArbTxTypeLocal::Eip1559,
            ArbTypedTransaction::Eip4844(_) => ArbTxTypeLocal::Eip4844,
            ArbTypedTransaction::Eip7702(_) => ArbTxTypeLocal::Eip7702,
            ArbTypedTransaction::Deposit(_) => ArbTxTypeLocal::Deposit,
            ArbTypedTransaction::Unsigned(_) => ArbTxTypeLocal::Unsigned,
            ArbTypedTransaction::Contract(_) => ArbTxTypeLocal::Contract,
            ArbTypedTransaction::Retry(_) => ArbTxTypeLocal::Retry,
            ArbTypedTransaction::SubmitRetryable(_) => ArbTxTypeLocal::SubmitRetryable,
            ArbTypedTransaction::Internal(_) => ArbTxTypeLocal::Internal,
        }
    }
}

/// Signed Arbitrum transaction with lazy hash caching.
#[derive(Clone, Debug, Eq)]
pub struct ArbTransactionSigned {
    hash: reth_primitives_traits::sync::OnceLock<TxHash>,
    signature: Signature,
    transaction: ArbTypedTransaction,
    input_cache: reth_primitives_traits::sync::OnceLock<Bytes>,
}

impl Deref for ArbTransactionSigned {
    type Target = ArbTypedTransaction;
    fn deref(&self) -> &Self::Target {
        &self.transaction
    }
}

impl ArbTransactionSigned {
    pub fn new(transaction: ArbTypedTransaction, signature: Signature, hash: B256) -> Self {
        Self {
            hash: hash.into(),
            signature,
            transaction,
            input_cache: Default::default(),
        }
    }

    pub fn new_unhashed(transaction: ArbTypedTransaction, signature: Signature) -> Self {
        Self {
            hash: Default::default(),
            signature,
            transaction,
            input_cache: Default::default(),
        }
    }

    /// Construct from a signed Ethereum envelope (standard tx types only).
    pub fn from_envelope(
        envelope: alloy_consensus::EthereumTxEnvelope<alloy_consensus::TxEip4844>,
    ) -> Self {
        use alloy_consensus::EthereumTxEnvelope;
        match envelope {
            EthereumTxEnvelope::Legacy(signed) => {
                let (tx, sig, hash) = signed.into_parts();
                Self::new(ArbTypedTransaction::Legacy(tx), sig, hash)
            }
            EthereumTxEnvelope::Eip2930(signed) => {
                let (tx, sig, hash) = signed.into_parts();
                Self::new(ArbTypedTransaction::Eip2930(tx), sig, hash)
            }
            EthereumTxEnvelope::Eip1559(signed) => {
                let (tx, sig, hash) = signed.into_parts();
                Self::new(ArbTypedTransaction::Eip1559(tx), sig, hash)
            }
            EthereumTxEnvelope::Eip4844(signed) => {
                let (tx, sig, hash) = signed.into_parts();
                Self::new(ArbTypedTransaction::Eip4844(tx), sig, hash)
            }
            EthereumTxEnvelope::Eip7702(signed) => {
                let (tx, sig, hash) = signed.into_parts();
                Self::new(ArbTypedTransaction::Eip7702(tx), sig, hash)
            }
        }
    }

    pub const fn signature(&self) -> &Signature {
        &self.signature
    }

    /// Returns the inner typed transaction.
    pub fn inner(&self) -> &ArbTypedTransaction {
        &self.transaction
    }

    /// Consume self and return (transaction, signature, hash).
    pub fn split(self) -> (ArbTypedTransaction, Signature, B256) {
        let hash = *self.hash.get_or_init(|| self.compute_hash());
        (self.transaction, self.signature, hash)
    }

    pub const fn tx_type(&self) -> ArbTxTypeLocal {
        match &self.transaction {
            ArbTypedTransaction::Deposit(_) => ArbTxTypeLocal::Deposit,
            ArbTypedTransaction::Unsigned(_) => ArbTxTypeLocal::Unsigned,
            ArbTypedTransaction::Contract(_) => ArbTxTypeLocal::Contract,
            ArbTypedTransaction::Retry(_) => ArbTxTypeLocal::Retry,
            ArbTypedTransaction::SubmitRetryable(_) => ArbTxTypeLocal::SubmitRetryable,
            ArbTypedTransaction::Internal(_) => ArbTxTypeLocal::Internal,
            ArbTypedTransaction::Legacy(_) => ArbTxTypeLocal::Legacy,
            ArbTypedTransaction::Eip2930(_) => ArbTxTypeLocal::Eip2930,
            ArbTypedTransaction::Eip1559(_) => ArbTxTypeLocal::Eip1559,
            ArbTypedTransaction::Eip4844(_) => ArbTxTypeLocal::Eip4844,
            ArbTypedTransaction::Eip7702(_) => ArbTxTypeLocal::Eip7702,
        }
    }

    fn compute_hash(&self) -> B256 {
        keccak256(self.encoded_2718())
    }

    fn zero_sig() -> Signature {
        Signature::new(U256::ZERO, U256::ZERO, false)
    }
}

// ---------------------------------------------------------------------------
// Hash / PartialEq — identity by tx hash
// ---------------------------------------------------------------------------

impl Hash for ArbTransactionSigned {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.tx_hash().hash(state)
    }
}

impl PartialEq for ArbTransactionSigned {
    fn eq(&self, other: &Self) -> bool {
        self.tx_hash() == other.tx_hash()
    }
}

impl InMemorySize for ArbTransactionSigned {
    fn size(&self) -> usize {
        core::mem::size_of::<TxHash>() + core::mem::size_of::<Signature>()
    }
}

// ---------------------------------------------------------------------------
// TxHashRef — lazy hash initialization
// ---------------------------------------------------------------------------

impl TxHashRef for ArbTransactionSigned {
    fn tx_hash(&self) -> &TxHash {
        self.hash.get_or_init(|| self.compute_hash())
    }
}

// ---------------------------------------------------------------------------
// SignedTransaction
// ---------------------------------------------------------------------------

impl SignedTransaction for ArbTransactionSigned {
    fn recalculate_hash(&self) -> B256 {
        keccak256(self.encoded_2718())
    }
}

// ---------------------------------------------------------------------------
// SignerRecoverable
// ---------------------------------------------------------------------------

impl alloy_consensus::transaction::SignerRecoverable for ArbTransactionSigned {
    fn recover_signer(
        &self,
    ) -> Result<Address, reth_primitives_traits::transaction::signed::RecoveryError> {
        match &self.transaction {
            // System tx types use the `from` field directly.
            ArbTypedTransaction::Deposit(tx) => Ok(tx.from),
            ArbTypedTransaction::Unsigned(tx) => Ok(tx.from),
            ArbTypedTransaction::Contract(tx) => Ok(tx.from),
            ArbTypedTransaction::Retry(tx) => Ok(tx.from),
            ArbTypedTransaction::SubmitRetryable(tx) => Ok(tx.from),
            ArbTypedTransaction::Internal(_) => Ok(ARBOS_ADDRESS),
            // Standard tx types use ECDSA recovery.
            ArbTypedTransaction::Legacy(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip2930(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip1559(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip4844(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip7702(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer(&self.signature, keccak256(&buf))
            }
        }
    }

    fn recover_signer_unchecked(
        &self,
    ) -> Result<Address, reth_primitives_traits::transaction::signed::RecoveryError> {
        match &self.transaction {
            ArbTypedTransaction::Deposit(tx) => Ok(tx.from),
            ArbTypedTransaction::Unsigned(tx) => Ok(tx.from),
            ArbTypedTransaction::Contract(tx) => Ok(tx.from),
            ArbTypedTransaction::Retry(tx) => Ok(tx.from),
            ArbTypedTransaction::SubmitRetryable(tx) => Ok(tx.from),
            ArbTypedTransaction::Internal(_) => Ok(ARBOS_ADDRESS),
            ArbTypedTransaction::Legacy(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer_unchecked(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip2930(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer_unchecked(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip1559(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer_unchecked(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip4844(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer_unchecked(&self.signature, keccak256(&buf))
            }
            ArbTypedTransaction::Eip7702(tx) => {
                let mut buf = Vec::new();
                tx.encode_for_signing(&mut buf);
                recover_signer_unchecked(&self.signature, keccak256(&buf))
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Typed2718
// ---------------------------------------------------------------------------

impl Typed2718 for ArbTransactionSigned {
    fn is_legacy(&self) -> bool {
        matches!(self.transaction, ArbTypedTransaction::Legacy(_))
    }

    fn ty(&self) -> u8 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(_) => 0u8,
            ArbTypedTransaction::Deposit(_) => ArbTxType::ArbitrumDepositTx.as_u8(),
            ArbTypedTransaction::Unsigned(_) => ArbTxType::ArbitrumUnsignedTx.as_u8(),
            ArbTypedTransaction::Contract(_) => ArbTxType::ArbitrumContractTx.as_u8(),
            ArbTypedTransaction::Retry(_) => ArbTxType::ArbitrumRetryTx.as_u8(),
            ArbTypedTransaction::SubmitRetryable(_) => ArbTxType::ArbitrumSubmitRetryableTx.as_u8(),
            ArbTypedTransaction::Internal(_) => ArbTxType::ArbitrumInternalTx.as_u8(),
            ArbTypedTransaction::Eip2930(_) => 0x01,
            ArbTypedTransaction::Eip1559(_) => 0x02,
            ArbTypedTransaction::Eip4844(_) => 0x03,
            ArbTypedTransaction::Eip7702(_) => 0x04,
        }
    }
}

// ---------------------------------------------------------------------------
// IsTyped2718
// ---------------------------------------------------------------------------

impl IsTyped2718 for ArbTransactionSigned {
    fn is_type(type_id: u8) -> bool {
        // Standard Ethereum types.
        matches!(type_id, 0x01 | 0x02 | 0x03 | 0x04) || ArbTxType::from_u8(type_id).is_ok()
    }
}

// ---------------------------------------------------------------------------
// Encodable2718
// ---------------------------------------------------------------------------

impl Encodable2718 for ArbTransactionSigned {
    fn type_flag(&self) -> Option<u8> {
        if self.is_legacy() {
            None
        } else {
            Some(self.ty())
        }
    }

    fn encode_2718_len(&self) -> usize {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.eip2718_encoded_length(&self.signature),
            ArbTypedTransaction::Deposit(tx) => tx.length() + 1,
            ArbTypedTransaction::Unsigned(tx) => tx.length() + 1,
            ArbTypedTransaction::Contract(tx) => tx.length() + 1,
            ArbTypedTransaction::Retry(tx) => tx.length() + 1,
            ArbTypedTransaction::SubmitRetryable(tx) => tx.length() + 1,
            ArbTypedTransaction::Internal(tx) => tx.length() + 1,
            ArbTypedTransaction::Eip2930(tx) => tx.eip2718_encoded_length(&self.signature),
            ArbTypedTransaction::Eip1559(tx) => tx.eip2718_encoded_length(&self.signature),
            ArbTypedTransaction::Eip4844(tx) => tx.eip2718_encoded_length(&self.signature),
            ArbTypedTransaction::Eip7702(tx) => tx.eip2718_encoded_length(&self.signature),
        }
    }

    fn encode_2718(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.eip2718_encode(&self.signature, out),
            ArbTypedTransaction::Deposit(tx) => {
                out.put_u8(ArbTxType::ArbitrumDepositTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::Unsigned(tx) => {
                out.put_u8(ArbTxType::ArbitrumUnsignedTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::Contract(tx) => {
                out.put_u8(ArbTxType::ArbitrumContractTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::Retry(tx) => {
                out.put_u8(ArbTxType::ArbitrumRetryTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::SubmitRetryable(tx) => {
                out.put_u8(ArbTxType::ArbitrumSubmitRetryableTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::Internal(tx) => {
                out.put_u8(ArbTxType::ArbitrumInternalTx.as_u8());
                tx.encode(out);
            }
            ArbTypedTransaction::Eip2930(tx) => tx.eip2718_encode(&self.signature, out),
            ArbTypedTransaction::Eip1559(tx) => tx.eip2718_encode(&self.signature, out),
            ArbTypedTransaction::Eip4844(tx) => tx.eip2718_encode(&self.signature, out),
            ArbTypedTransaction::Eip7702(tx) => tx.eip2718_encode(&self.signature, out),
        }
    }
}

// ---------------------------------------------------------------------------
// Decodable2718
// ---------------------------------------------------------------------------

impl Decodable2718 for ArbTransactionSigned {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> Eip2718Result<Self> {
        // Try Arbitrum-specific types first.
        if let Ok(kind) = ArbTxType::from_u8(ty) {
            return Ok(match kind {
                ArbTxType::ArbitrumDepositTx => {
                    let tx = ArbDepositTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::Deposit(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumUnsignedTx => {
                    let tx = ArbUnsignedTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::Unsigned(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumContractTx => {
                    let tx = ArbContractTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::Contract(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumRetryTx => {
                    let tx = ArbRetryTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::Retry(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumSubmitRetryableTx => {
                    let tx = ArbSubmitRetryableTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::SubmitRetryable(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumInternalTx => {
                    let tx = ArbInternalTx::decode(buf)?;
                    Self::new_unhashed(ArbTypedTransaction::Internal(tx), Self::zero_sig())
                }
                ArbTxType::ArbitrumLegacyTx => return Err(Eip2718Error::UnexpectedType(0x78)),
            });
        }

        // Standard Ethereum typed transactions.
        match alloy_consensus::TxType::try_from(ty).map_err(|_| Eip2718Error::UnexpectedType(ty))? {
            alloy_consensus::TxType::Legacy => Err(Eip2718Error::UnexpectedType(0)),
            alloy_consensus::TxType::Eip2930 => {
                let (tx, sig) = alloy_consensus::TxEip2930::rlp_decode_with_signature(buf)?;
                Ok(Self::new_unhashed(ArbTypedTransaction::Eip2930(tx), sig))
            }
            alloy_consensus::TxType::Eip1559 => {
                let (tx, sig) = alloy_consensus::TxEip1559::rlp_decode_with_signature(buf)?;
                Ok(Self::new_unhashed(ArbTypedTransaction::Eip1559(tx), sig))
            }
            alloy_consensus::TxType::Eip4844 => {
                let (tx, sig) = alloy_consensus::TxEip4844::rlp_decode_with_signature(buf)?;
                Ok(Self::new_unhashed(ArbTypedTransaction::Eip4844(tx), sig))
            }
            alloy_consensus::TxType::Eip7702 => {
                let (tx, sig) = alloy_consensus::TxEip7702::rlp_decode_with_signature(buf)?;
                Ok(Self::new_unhashed(ArbTypedTransaction::Eip7702(tx), sig))
            }
        }
    }

    fn fallback_decode(buf: &mut &[u8]) -> Eip2718Result<Self> {
        let (tx, sig, hash) = TxLegacy::rlp_decode_signed(buf)?.into_parts();
        let signed_tx = Self::new_unhashed(ArbTypedTransaction::Legacy(tx), sig);
        signed_tx.hash.get_or_init(|| hash);
        Ok(signed_tx)
    }
}

// ---------------------------------------------------------------------------
// Encodable / Decodable (RLP network encoding)
// ---------------------------------------------------------------------------

impl Encodable for ArbTransactionSigned {
    fn encode(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        self.network_encode(out);
    }
    fn length(&self) -> usize {
        let mut payload_length = self.encode_2718_len();
        if !self.is_legacy() {
            payload_length += alloy_rlp::Header {
                list: false,
                payload_length,
            }
            .length();
        }
        payload_length
    }
}

impl Decodable for ArbTransactionSigned {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::network_decode(buf).map_err(Into::into)
    }
}

// ---------------------------------------------------------------------------
// Transaction (alloy_consensus::Transaction)
// ---------------------------------------------------------------------------

impl ConsensusTx for ArbTransactionSigned {
    fn chain_id(&self) -> Option<u64> {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.chain_id.map(|id| id as u64),
            ArbTypedTransaction::Deposit(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::Unsigned(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::Contract(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::Retry(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::SubmitRetryable(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::Internal(tx) => Some(tx.chain_id.to::<u64>()),
            ArbTypedTransaction::Eip2930(tx) => Some(tx.chain_id as u64),
            ArbTypedTransaction::Eip1559(tx) => Some(tx.chain_id as u64),
            ArbTypedTransaction::Eip4844(tx) => Some(tx.chain_id as u64),
            ArbTypedTransaction::Eip7702(tx) => Some(tx.chain_id as u64),
        }
    }

    fn nonce(&self) -> u64 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.nonce,
            ArbTypedTransaction::Deposit(_) => 0,
            ArbTypedTransaction::Unsigned(tx) => tx.nonce,
            ArbTypedTransaction::Contract(_) => 0,
            ArbTypedTransaction::Retry(tx) => tx.nonce,
            ArbTypedTransaction::SubmitRetryable(_) => 0,
            ArbTypedTransaction::Internal(_) => 0,
            ArbTypedTransaction::Eip2930(tx) => tx.nonce,
            ArbTypedTransaction::Eip1559(tx) => tx.nonce,
            ArbTypedTransaction::Eip4844(tx) => tx.nonce,
            ArbTypedTransaction::Eip7702(tx) => tx.nonce,
        }
    }

    fn gas_limit(&self) -> u64 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.gas_limit,
            ArbTypedTransaction::Deposit(_) => 0,
            ArbTypedTransaction::Unsigned(tx) => tx.gas,
            ArbTypedTransaction::Contract(tx) => tx.gas,
            ArbTypedTransaction::Retry(tx) => tx.gas,
            ArbTypedTransaction::SubmitRetryable(tx) => tx.gas,
            ArbTypedTransaction::Internal(_) => 0,
            ArbTypedTransaction::Eip2930(tx) => tx.gas_limit,
            ArbTypedTransaction::Eip1559(tx) => tx.gas_limit,
            ArbTypedTransaction::Eip4844(tx) => tx.gas_limit,
            ArbTypedTransaction::Eip7702(tx) => tx.gas_limit,
        }
    }

    fn gas_price(&self) -> Option<u128> {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => Some(tx.gas_price.into()),
            ArbTypedTransaction::Eip2930(tx) => Some(tx.gas_price.into()),
            _ => None,
        }
    }

    fn max_fee_per_gas(&self) -> u128 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.gas_price.into(),
            ArbTypedTransaction::Eip2930(tx) => tx.gas_price.into(),
            ArbTypedTransaction::Unsigned(tx) => tx.gas_fee_cap.to::<u128>(),
            ArbTypedTransaction::Contract(tx) => tx.gas_fee_cap.to::<u128>(),
            ArbTypedTransaction::Retry(tx) => tx.gas_fee_cap.to::<u128>(),
            ArbTypedTransaction::SubmitRetryable(tx) => tx.gas_fee_cap.to::<u128>(),
            ArbTypedTransaction::Eip1559(tx) => tx.max_fee_per_gas as u128,
            ArbTypedTransaction::Eip4844(tx) => tx.max_fee_per_gas as u128,
            ArbTypedTransaction::Eip7702(tx) => tx.max_fee_per_gas as u128,
            _ => 0,
        }
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        Some(0)
    }

    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        Some(0)
    }

    fn priority_fee_or_price(&self) -> u128 {
        self.gas_price().unwrap_or(0)
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.gas_price.into(),
            ArbTypedTransaction::Eip2930(tx) => tx.gas_price.into(),
            // All other types: effective_gas_price = basefee (DropTip behavior).
            _ => base_fee.unwrap_or(0) as u128,
        }
    }

    fn effective_tip_per_gas(&self, _base_fee: u64) -> Option<u128> {
        Some(0)
    }

    fn is_dynamic_fee(&self) -> bool {
        !matches!(
            self.transaction,
            ArbTypedTransaction::Legacy(_) | ArbTypedTransaction::Eip2930(_)
        )
    }

    fn kind(&self) -> TxKind {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.to,
            ArbTypedTransaction::Deposit(tx) => {
                if tx.to == Address::ZERO {
                    TxKind::Create
                } else {
                    TxKind::Call(tx.to)
                }
            }
            ArbTypedTransaction::Unsigned(tx) => match tx.to {
                Some(to) => TxKind::Call(to),
                None => TxKind::Create,
            },
            ArbTypedTransaction::Contract(tx) => match tx.to {
                Some(to) => TxKind::Call(to),
                None => TxKind::Create,
            },
            ArbTypedTransaction::Retry(tx) => match tx.to {
                Some(to) => TxKind::Call(to),
                None => TxKind::Create,
            },
            ArbTypedTransaction::SubmitRetryable(_) => TxKind::Call(RETRYABLE_ADDRESS),
            ArbTypedTransaction::Internal(_) => TxKind::Call(ARBOS_ADDRESS),
            ArbTypedTransaction::Eip2930(tx) => tx.to,
            ArbTypedTransaction::Eip1559(tx) => tx.to,
            ArbTypedTransaction::Eip4844(tx) => TxKind::Call(tx.to),
            ArbTypedTransaction::Eip7702(tx) => TxKind::Call(tx.to),
        }
    }

    fn is_create(&self) -> bool {
        matches!(self.kind(), TxKind::Create)
    }

    fn value(&self) -> U256 {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => tx.value,
            ArbTypedTransaction::Deposit(tx) => tx.value,
            ArbTypedTransaction::Unsigned(tx) => tx.value,
            ArbTypedTransaction::Contract(tx) => tx.value,
            ArbTypedTransaction::Retry(tx) => tx.value,
            ArbTypedTransaction::SubmitRetryable(tx) => tx.retry_value,
            ArbTypedTransaction::Internal(_) => U256::ZERO,
            ArbTypedTransaction::Eip2930(tx) => tx.value,
            ArbTypedTransaction::Eip1559(tx) => tx.value,
            ArbTypedTransaction::Eip4844(tx) => tx.value,
            ArbTypedTransaction::Eip7702(tx) => tx.value,
        }
    }

    fn input(&self) -> &Bytes {
        match &self.transaction {
            ArbTypedTransaction::Legacy(tx) => &tx.input,
            ArbTypedTransaction::Deposit(_) => self.input_cache.get_or_init(Bytes::new),
            ArbTypedTransaction::Unsigned(tx) => self
                .input_cache
                .get_or_init(|| Bytes::from(tx.data.clone())),
            ArbTypedTransaction::Contract(tx) => self
                .input_cache
                .get_or_init(|| Bytes::from(tx.data.clone())),
            ArbTypedTransaction::Retry(tx) => self
                .input_cache
                .get_or_init(|| Bytes::from(tx.data.clone())),
            ArbTypedTransaction::SubmitRetryable(tx) => self.input_cache.get_or_init(|| {
                let sel = arb_alloy_predeploys::selector(
                    arb_alloy_predeploys::SIG_RETRY_SUBMIT_RETRYABLE,
                );
                let mut out = Vec::with_capacity(4 + tx.retry_data.len());
                out.extend_from_slice(&sel);
                out.extend_from_slice(&tx.retry_data);
                Bytes::from(out)
            }),
            ArbTypedTransaction::Internal(tx) => self
                .input_cache
                .get_or_init(|| Bytes::from(tx.data.clone())),
            ArbTypedTransaction::Eip2930(tx) => &tx.input,
            ArbTypedTransaction::Eip1559(tx) => &tx.input,
            ArbTypedTransaction::Eip4844(tx) => &tx.input,
            ArbTypedTransaction::Eip7702(tx) => &tx.input,
        }
    }

    fn access_list(&self) -> Option<&alloy_eips::eip2930::AccessList> {
        match &self.transaction {
            ArbTypedTransaction::Eip2930(tx) => Some(&tx.access_list),
            ArbTypedTransaction::Eip1559(tx) => Some(&tx.access_list),
            ArbTypedTransaction::Eip4844(tx) => Some(&tx.access_list),
            ArbTypedTransaction::Eip7702(tx) => Some(&tx.access_list),
            _ => None,
        }
    }

    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        None
    }

    fn authorization_list(&self) -> Option<&[alloy_eips::eip7702::SignedAuthorization]> {
        None
    }
}

// ---------------------------------------------------------------------------
// serde — serialize via 2718 encoding
// ---------------------------------------------------------------------------

impl serde::Serialize for ArbTransactionSigned {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut state = serializer.serialize_struct("ArbTransactionSigned", 2)?;
        state.serialize_field("signature", &self.signature)?;
        state.serialize_field("hash", self.tx_hash())?;
        state.end()
    }
}

impl<'de> serde::Deserialize<'de> for ArbTransactionSigned {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        #[derive(serde::Deserialize)]
        struct Helper {
            signature: Signature,
            #[serde(default)]
            transaction_encoded_2718: Option<alloy_primitives::Bytes>,
        }
        let helper = Helper::deserialize(deserializer)?;
        if let Some(encoded) = helper.transaction_encoded_2718 {
            let mut slice: &[u8] = encoded.as_ref();
            let parsed = Self::network_decode(&mut slice).map_err(serde::de::Error::custom)?;
            Ok(parsed)
        } else {
            // Fallback: return a default-like empty tx (legacy with zero fields).
            Ok(Self::new_unhashed(
                ArbTypedTransaction::Legacy(TxLegacy::default()),
                helper.signature,
            ))
        }
    }
}

// ---------------------------------------------------------------------------
// RlpBincode — required by SerdeBincodeCompat
// ---------------------------------------------------------------------------

impl reth_primitives_traits::serde_bincode_compat::RlpBincode for ArbTransactionSigned {}

// ---------------------------------------------------------------------------
// Compact — required by MaybeCompact when reth-codec feature is active
// ---------------------------------------------------------------------------

impl reth_codecs::Compact for ArbTransactionSigned {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        // Simple approach: encode via 2718 and prefix with length.
        let encoded = self.encoded_2718();
        let len = encoded.len() as u32;
        buf.put_u32(len);
        buf.put_slice(&encoded);
        // Signature
        let sig_bytes = self.signature.as_bytes();
        buf.put_slice(&sig_bytes);
        0
    }

    fn from_compact(buf: &[u8], _len: usize) -> (Self, &[u8]) {
        use bytes::Buf;
        let mut slice = buf;
        let tx_len = slice.get_u32() as usize;
        let tx_bytes = &slice[..tx_len];
        slice = &slice[tx_len..];

        let mut tx_buf = tx_bytes;
        let tx = Self::network_decode(&mut tx_buf).unwrap_or_else(|_| {
            Self::new_unhashed(
                ArbTypedTransaction::Legacy(TxLegacy::default()),
                Signature::new(U256::ZERO, U256::ZERO, false),
            )
        });

        // Read signature (65 bytes)
        if slice.len() >= 65 {
            let _sig_bytes = &slice[..65];
            slice = &slice[65..];
        }

        (tx, slice)
    }
}

// ---------------------------------------------------------------------------
// Compress / Decompress — delegates to Compact for database storage
// ---------------------------------------------------------------------------

impl reth_db_api::table::Compress for ArbTransactionSigned {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let _ = reth_codecs::Compact::to_compact(self, buf);
    }
}

impl reth_db_api::table::Decompress for ArbTransactionSigned {
    fn decompress(value: &[u8]) -> Result<Self, reth_db_api::DatabaseError> {
        let (obj, _) = reth_codecs::Compact::from_compact(value, value.len());
        Ok(obj)
    }
}

// ---------------------------------------------------------------------------
// Arbitrum transaction data extraction
// ---------------------------------------------------------------------------

/// Data extracted from a SubmitRetryable transaction for processing.
#[derive(Debug, Clone)]
pub struct SubmitRetryableInfo {
    pub from: Address,
    pub deposit_value: U256,
    pub retry_value: U256,
    pub gas_fee_cap: U256,
    pub gas: u64,
    pub retry_to: Option<Address>,
    pub retry_data: Vec<u8>,
    pub beneficiary: Address,
    pub max_submission_fee: U256,
    pub fee_refund_addr: Address,
    pub l1_base_fee: U256,
    pub request_id: B256,
}

/// Data extracted from a RetryTx transaction for processing.
#[derive(Debug, Clone)]
pub struct RetryTxInfo {
    pub from: Address,
    pub ticket_id: B256,
    pub refund_to: Address,
    pub gas_fee_cap: U256,
    pub max_refund: U256,
    pub submission_fee_refund: U256,
}

/// Trait for extracting Arbitrum-specific transaction data beyond the
/// standard `Transaction` trait.
pub trait ArbTransactionExt {
    fn submit_retryable_info(&self) -> Option<SubmitRetryableInfo> {
        None
    }
    fn retry_tx_info(&self) -> Option<RetryTxInfo> {
        None
    }
}

impl ArbTransactionExt for ArbTransactionSigned {
    fn submit_retryable_info(&self) -> Option<SubmitRetryableInfo> {
        match &self.transaction {
            ArbTypedTransaction::SubmitRetryable(tx) => Some(SubmitRetryableInfo {
                from: tx.from,
                deposit_value: tx.deposit_value,
                retry_value: tx.retry_value,
                gas_fee_cap: tx.gas_fee_cap,
                gas: tx.gas,
                retry_to: tx.retry_to,
                retry_data: tx.retry_data.to_vec(),
                beneficiary: tx.beneficiary,
                max_submission_fee: tx.max_submission_fee,
                fee_refund_addr: tx.fee_refund_addr,
                l1_base_fee: tx.l1_base_fee,
                request_id: tx.request_id,
            }),
            _ => None,
        }
    }

    fn retry_tx_info(&self) -> Option<RetryTxInfo> {
        match &self.transaction {
            ArbTypedTransaction::Retry(tx) => Some(RetryTxInfo {
                from: tx.from,
                ticket_id: tx.ticket_id,
                refund_to: tx.refund_to,
                gas_fee_cap: tx.gas_fee_cap,
                max_refund: tx.max_refund,
                submission_fee_refund: tx.submission_fee_refund,
            }),
            _ => None,
        }
    }
}

/// Standard Ethereum transaction envelopes don't carry retryable data.
impl<T> ArbTransactionExt for alloy_consensus::EthereumTxEnvelope<T> {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn roundtrip_unsigned_tx() {
        let tx = ArbUnsignedTx {
            chain_id: U256::from(42161u64),
            from: alloy_primitives::address!("00000000000000000000000000000000000000aa"),
            nonce: 7,
            gas_fee_cap: U256::from(1_000_000u64),
            gas: 21000,
            to: Some(alloy_primitives::address!(
                "00000000000000000000000000000000000000bb"
            )),
            value: U256::from(123u64),
            data: Vec::new().into(),
        };

        let mut enc = Vec::with_capacity(1 + tx.length());
        enc.push(ArbTxType::ArbitrumUnsignedTx.as_u8());
        tx.encode(&mut enc);

        let signed =
            ArbTransactionSigned::decode_2718_exact(enc.as_slice()).expect("typed decode ok");
        assert_eq!(signed.tx_type(), ArbTxTypeLocal::Unsigned);
        assert_eq!(signed.chain_id(), Some(42161));
        assert_eq!(signed.nonce(), 7);
        assert_eq!(signed.gas_limit(), 21000);
        assert_eq!(signed.value(), U256::from(123u64));
    }

    #[test]
    fn deposit_tx_has_zero_gas() {
        let tx = ArbDepositTx {
            chain_id: U256::from(42161u64),
            l1_request_id: B256::ZERO,
            from: Address::ZERO,
            to: Address::ZERO,
            value: U256::from(100u64),
        };

        let signed = ArbTransactionSigned::new_unhashed(
            ArbTypedTransaction::Deposit(tx),
            ArbTransactionSigned::zero_sig(),
        );

        assert_eq!(signed.gas_limit(), 0);
        assert_eq!(signed.nonce(), 0);
        assert_eq!(signed.tx_type(), ArbTxTypeLocal::Deposit);
    }
}
