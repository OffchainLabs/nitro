use alloc::vec::Vec;

use alloy_consensus::{
    Eip2718EncodableReceipt, Eip658Value, Receipt as AlloyReceipt, TxReceipt, Typed2718,
};
use alloy_eips::{Decodable2718, Encodable2718};
use alloy_primitives::{Bloom, Log};
use alloy_rlp::{Decodable, Encodable};
use arb_alloy_consensus::tx::ArbTxType;
use reth_primitives_traits::InMemorySize;

use crate::multigas::MultiGas;

/// Arbitrum receipt: wraps the per-type receipt kind with L1 gas metadata.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ArbReceipt {
    pub kind: ArbReceiptKind,
    /// Gas units used for L1 calldata posting (poster gas).
    /// Populated by the block executor after receipt construction.
    pub gas_used_for_l1: u64,
    /// Multi-dimensional gas usage breakdown.
    /// Populated when multi-gas tracking is enabled (ArbOS v60+).
    pub multi_gas_used: MultiGas,
}

impl ArbReceipt {
    /// Create a new receipt with no L1 gas usage (filled in later).
    pub fn new(kind: ArbReceiptKind) -> Self {
        Self {
            kind,
            gas_used_for_l1: 0,
            multi_gas_used: MultiGas::zero(),
        }
    }

    pub fn with_gas_used_for_l1(mut self, gas: u64) -> Self {
        self.gas_used_for_l1 = gas;
        self
    }
}

/// Trait for setting Arbitrum-specific fields on a receipt after construction.
pub trait SetArbReceiptFields {
    fn set_gas_used_for_l1(&mut self, gas: u64);
    fn set_multi_gas_used(&mut self, multi_gas: MultiGas);
}

impl SetArbReceiptFields for ArbReceipt {
    fn set_gas_used_for_l1(&mut self, gas: u64) {
        self.gas_used_for_l1 = gas;
    }

    fn set_multi_gas_used(&mut self, multi_gas: MultiGas) {
        self.multi_gas_used = multi_gas;
    }
}

/// Per-type receipt variants matching the Arbitrum transaction type.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ArbReceiptKind {
    Legacy(AlloyReceipt),
    Eip1559(AlloyReceipt),
    Eip2930(AlloyReceipt),
    Eip7702(AlloyReceipt),
    Deposit(ArbDepositReceipt),
    Unsigned(AlloyReceipt),
    Contract(AlloyReceipt),
    Retry(AlloyReceipt),
    SubmitRetryable(AlloyReceipt),
    Internal(AlloyReceipt),
}

/// Deposit receipts always succeed with no gas and no logs.
#[derive(Clone, Debug, PartialEq, Eq, Default)]
pub struct ArbDepositReceipt;

// ---------------------------------------------------------------------------
// ArbReceiptKind — inherent methods (encoding internals)
// ---------------------------------------------------------------------------

impl ArbReceiptKind {
    pub const fn arb_tx_type(&self) -> ArbTxType {
        match self {
            Self::Legacy(_) | Self::Eip2930(_) | Self::Eip1559(_) | Self::Eip7702(_) => {
                ArbTxType::ArbitrumLegacyTx
            }
            Self::Deposit(_) => ArbTxType::ArbitrumDepositTx,
            Self::Unsigned(_) => ArbTxType::ArbitrumUnsignedTx,
            Self::Contract(_) => ArbTxType::ArbitrumContractTx,
            Self::Retry(_) => ArbTxType::ArbitrumRetryTx,
            Self::SubmitRetryable(_) => ArbTxType::ArbitrumSubmitRetryableTx,
            Self::Internal(_) => ArbTxType::ArbitrumInternalTx,
        }
    }

    /// Returns the underlying `AlloyReceipt` for variants that wrap one.
    ///
    /// Returns `None` for [`ArbReceiptKind::Deposit`], which has no inner
    /// receipt body (deposits always succeed with zero gas and no logs).
    pub const fn as_receipt(&self) -> Option<&AlloyReceipt> {
        match self {
            Self::Legacy(r)
            | Self::Eip2930(r)
            | Self::Eip1559(r)
            | Self::Eip7702(r)
            | Self::Unsigned(r)
            | Self::Contract(r)
            | Self::Retry(r)
            | Self::SubmitRetryable(r)
            | Self::Internal(r) => Some(r),
            Self::Deposit(_) => None,
        }
    }

    fn rlp_encoded_fields_length(&self, bloom: &Bloom) -> usize {
        match self {
            Self::Legacy(r)
            | Self::Eip2930(r)
            | Self::Eip1559(r)
            | Self::Eip7702(r)
            | Self::Unsigned(r)
            | Self::Contract(r)
            | Self::Retry(r)
            | Self::SubmitRetryable(r)
            | Self::Internal(r) => r.rlp_encoded_fields_length_with_bloom(bloom),
            Self::Deposit(_) => {
                Eip658Value::Eip658(true).length()
                    + 0u64.length()
                    + bloom.length()
                    + Vec::<Log>::new().length()
            }
        }
    }

    fn rlp_encode_fields(&self, bloom: &Bloom, out: &mut dyn alloy_rlp::bytes::BufMut) {
        match self {
            Self::Legacy(r)
            | Self::Eip2930(r)
            | Self::Eip1559(r)
            | Self::Eip7702(r)
            | Self::Unsigned(r)
            | Self::Contract(r)
            | Self::Retry(r)
            | Self::SubmitRetryable(r)
            | Self::Internal(r) => r.rlp_encode_fields_with_bloom(bloom, out),
            Self::Deposit(_) => {
                Eip658Value::Eip658(true).encode(out);
                (0u64).encode(out);
                bloom.encode(out);
                let logs: Vec<Log> = Vec::new();
                logs.encode(out);
            }
        }
    }

    fn rlp_header_inner(&self, bloom: &Bloom) -> alloy_rlp::Header {
        alloy_rlp::Header {
            list: true,
            payload_length: self.rlp_encoded_fields_length(bloom),
        }
    }

    fn rlp_encode_fields_without_bloom(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        match self {
            Self::Legacy(r)
            | Self::Eip2930(r)
            | Self::Eip1559(r)
            | Self::Eip7702(r)
            | Self::Unsigned(r)
            | Self::Contract(r)
            | Self::Retry(r)
            | Self::SubmitRetryable(r)
            | Self::Internal(r) => {
                r.status.encode(out);
                r.cumulative_gas_used.encode(out);
                r.logs.encode(out);
            }
            Self::Deposit(_) => {
                Eip658Value::Eip658(true).encode(out);
                (0u64).encode(out);
                let logs: Vec<Log> = Vec::new();
                logs.encode(out);
            }
        }
    }

    fn rlp_encoded_fields_length_without_bloom(&self) -> usize {
        match self {
            Self::Legacy(r)
            | Self::Eip2930(r)
            | Self::Eip1559(r)
            | Self::Eip7702(r)
            | Self::Unsigned(r)
            | Self::Contract(r)
            | Self::Retry(r)
            | Self::SubmitRetryable(r)
            | Self::Internal(r) => {
                r.status.length() + r.cumulative_gas_used.length() + r.logs.length()
            }
            Self::Deposit(_) => {
                Eip658Value::Eip658(true).length() + (0u64).length() + Vec::<Log>::new().length()
            }
        }
    }

    fn rlp_header_inner_without_bloom(&self) -> alloy_rlp::Header {
        alloy_rlp::Header {
            list: true,
            payload_length: self.rlp_encoded_fields_length_without_bloom(),
        }
    }

    fn rlp_decode_inner(
        buf: &mut &[u8],
        tx_type: ArbTxType,
    ) -> alloy_rlp::Result<alloy_consensus::ReceiptWithBloom<ArbReceipt>> {
        match tx_type {
            ArbTxType::ArbitrumDepositTx => {
                let header = alloy_rlp::Header::decode(buf)?;
                if !header.list {
                    return Err(alloy_rlp::Error::UnexpectedString);
                }
                let remaining = buf.len();
                let _status: Eip658Value = alloy_rlp::Decodable::decode(buf)?;
                let _cumu: u64 = alloy_rlp::Decodable::decode(buf)?;
                let logs_bloom: Bloom = alloy_rlp::Decodable::decode(buf)?;
                let _logs: Vec<Log> = alloy_rlp::Decodable::decode(buf)?;
                if buf.len() + header.payload_length != remaining {
                    return Err(alloy_rlp::Error::UnexpectedLength);
                }
                Ok(alloy_consensus::ReceiptWithBloom {
                    receipt: ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt)),
                    logs_bloom,
                })
            }
            _ => {
                let alloy_consensus::ReceiptWithBloom {
                    receipt,
                    logs_bloom,
                } = <AlloyReceipt as alloy_consensus::RlpDecodableReceipt>::rlp_decode_with_bloom(
                    buf,
                )?;
                Ok(alloy_consensus::ReceiptWithBloom {
                    receipt: ArbReceipt::new(ArbReceiptKind::Legacy(receipt)),
                    logs_bloom,
                })
            }
        }
    }

    fn rlp_decode_inner_without_bloom(
        buf: &mut &[u8],
        tx_type: ArbTxType,
    ) -> alloy_rlp::Result<ArbReceipt> {
        let header = alloy_rlp::Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }
        let remaining = buf.len();
        let status: Eip658Value = alloy_rlp::Decodable::decode(buf)?;
        let cumulative_gas_used: u64 = alloy_rlp::Decodable::decode(buf)?;
        let logs: Vec<Log> = alloy_rlp::Decodable::decode(buf)?;
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        let receipt = AlloyReceipt {
            status,
            cumulative_gas_used,
            logs,
        };
        let kind = match tx_type {
            ArbTxType::ArbitrumDepositTx => ArbReceiptKind::Deposit(ArbDepositReceipt),
            ArbTxType::ArbitrumUnsignedTx => ArbReceiptKind::Unsigned(receipt),
            ArbTxType::ArbitrumContractTx => ArbReceiptKind::Contract(receipt),
            ArbTxType::ArbitrumRetryTx => ArbReceiptKind::Retry(receipt),
            ArbTxType::ArbitrumSubmitRetryableTx => ArbReceiptKind::SubmitRetryable(receipt),
            ArbTxType::ArbitrumInternalTx => ArbReceiptKind::Internal(receipt),
            ArbTxType::ArbitrumLegacyTx => ArbReceiptKind::Legacy(receipt),
        };
        Ok(ArbReceipt::new(kind))
    }
}

// ---------------------------------------------------------------------------
// InMemorySize
// ---------------------------------------------------------------------------

impl InMemorySize for ArbReceipt {
    fn size(&self) -> usize {
        core::mem::size_of::<u64>() // gas_used_for_l1
            + core::mem::size_of::<MultiGas>() // multi_gas_used
    }
}

// ---------------------------------------------------------------------------
// TxReceipt — delegate to kind
// ---------------------------------------------------------------------------

impl TxReceipt for ArbReceipt {
    type Log = Log;

    fn status_or_post_state(&self) -> Eip658Value {
        match self.kind.as_receipt() {
            Some(r) => r.status_or_post_state(),
            None => Eip658Value::Eip658(true),
        }
    }

    fn status(&self) -> bool {
        match self.kind.as_receipt() {
            Some(r) => r.status(),
            None => true,
        }
    }

    fn bloom(&self) -> Bloom {
        match self.kind.as_receipt() {
            Some(r) => r.bloom(),
            None => Bloom::ZERO,
        }
    }

    fn cumulative_gas_used(&self) -> u64 {
        match self.kind.as_receipt() {
            Some(r) => r.cumulative_gas_used(),
            None => 0,
        }
    }

    fn logs(&self) -> &[Self::Log] {
        match self.kind.as_receipt() {
            Some(r) => r.logs(),
            None => &[],
        }
    }

    fn into_logs(self) -> Vec<Self::Log> {
        match self.kind {
            ArbReceiptKind::Legacy(r)
            | ArbReceiptKind::Eip2930(r)
            | ArbReceiptKind::Eip1559(r)
            | ArbReceiptKind::Eip7702(r)
            | ArbReceiptKind::Unsigned(r)
            | ArbReceiptKind::Contract(r)
            | ArbReceiptKind::Retry(r)
            | ArbReceiptKind::SubmitRetryable(r)
            | ArbReceiptKind::Internal(r) => r.logs,
            ArbReceiptKind::Deposit(_) => Vec::new(),
        }
    }
}

// ---------------------------------------------------------------------------
// Typed2718 — delegate to kind
// ---------------------------------------------------------------------------

impl Typed2718 for ArbReceipt {
    fn is_legacy(&self) -> bool {
        matches!(self.kind, ArbReceiptKind::Legacy(_))
    }

    fn ty(&self) -> u8 {
        match &self.kind {
            ArbReceiptKind::Legacy(_) => 0x00,
            ArbReceiptKind::Eip2930(_) => 0x01,
            ArbReceiptKind::Eip1559(_) => 0x02,
            ArbReceiptKind::Eip7702(_) => 0x04,
            ArbReceiptKind::Deposit(_) => 0x64,
            ArbReceiptKind::Unsigned(_) => 0x65,
            ArbReceiptKind::Contract(_) => 0x66,
            ArbReceiptKind::Retry(_) => 0x68,
            ArbReceiptKind::SubmitRetryable(_) => 0x69,
            ArbReceiptKind::Internal(_) => 0x6A,
        }
    }
}

// ---------------------------------------------------------------------------
// Eip2718EncodableReceipt — consensus encoding (no gas_used_for_l1)
// ---------------------------------------------------------------------------

impl Eip2718EncodableReceipt for ArbReceipt {
    fn eip2718_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        let inner_len = self.kind.rlp_header_inner(bloom).length_with_payload();
        if !self.is_legacy() {
            1 + inner_len
        } else {
            inner_len
        }
    }

    fn eip2718_encode_with_bloom(&self, bloom: &Bloom, out: &mut dyn alloy_rlp::bytes::BufMut) {
        if !self.is_legacy() {
            out.put_u8(self.ty());
        }
        self.kind.rlp_header_inner(bloom).encode(out);
        self.kind.rlp_encode_fields(bloom, out);
    }
}

// ---------------------------------------------------------------------------
// RlpEncodableReceipt
// ---------------------------------------------------------------------------

impl alloy_consensus::RlpEncodableReceipt for ArbReceipt {
    fn rlp_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        let mut len = self.eip2718_encoded_length_with_bloom(bloom);
        if !self.is_legacy() {
            len += alloy_rlp::Header {
                list: false,
                payload_length: self.eip2718_encoded_length_with_bloom(bloom),
            }
            .length();
        }
        len
    }

    fn rlp_encode_with_bloom(&self, bloom: &Bloom, out: &mut dyn alloy_rlp::bytes::BufMut) {
        if !self.is_legacy() {
            alloy_rlp::Header {
                list: false,
                payload_length: self.eip2718_encoded_length_with_bloom(bloom),
            }
            .encode(out);
        }
        self.eip2718_encode_with_bloom(bloom, out);
    }
}

// ---------------------------------------------------------------------------
// RlpDecodableReceipt
// ---------------------------------------------------------------------------

impl alloy_consensus::RlpDecodableReceipt for ArbReceipt {
    fn rlp_decode_with_bloom(
        buf: &mut &[u8],
    ) -> alloy_rlp::Result<alloy_consensus::ReceiptWithBloom<Self>> {
        let header_buf = &mut &**buf;
        let header = alloy_rlp::Header::decode(header_buf)?;
        if header.list {
            return ArbReceiptKind::rlp_decode_inner(buf, ArbTxType::ArbitrumLegacyTx);
        }
        *buf = *header_buf;
        let remaining = buf.len();
        let ty = u8::decode(buf)?;
        let tx_type = ArbTxType::from_u8(ty)
            .map_err(|_| alloy_rlp::Error::Custom("unexpected arb receipt tx type"))?;
        let this = ArbReceiptKind::rlp_decode_inner(buf, tx_type)?;
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(this)
    }
}

// ---------------------------------------------------------------------------
// Encodable2718 / Decodable2718
// ---------------------------------------------------------------------------

impl Encodable2718 for ArbReceipt {
    fn encode_2718_len(&self) -> usize {
        let type_len = if self.is_legacy() { 0 } else { 1 };
        type_len
            + self
                .kind
                .rlp_header_inner_without_bloom()
                .length_with_payload()
    }

    fn encode_2718(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        if !self.is_legacy() {
            out.put_u8(self.ty());
        }
        self.kind.rlp_header_inner_without_bloom().encode(out);
        self.kind.rlp_encode_fields_without_bloom(out);
    }
}

impl Decodable2718 for ArbReceipt {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> alloy_eips::eip2718::Eip2718Result<Self> {
        // Standard EVM types use their raw type byte in 2718 encoding but map to
        // specific ArbReceiptKind variants. ArbTxType only covers Arbitrum-specific
        // types (0x64+), so handle standard EVM types here first.
        match ty {
            0x01 => {
                return Self::decode_standard_receipt(buf, ArbReceiptKind::Eip2930);
            }
            0x02 => {
                return Self::decode_standard_receipt(buf, ArbReceiptKind::Eip1559);
            }
            0x04 => {
                return Self::decode_standard_receipt(buf, ArbReceiptKind::Eip7702);
            }
            _ => {}
        }
        let tx_type = ArbTxType::from_u8(ty)
            .map_err(|_| alloy_eips::eip2718::Eip2718Error::UnexpectedType(ty))?;
        Ok(ArbReceiptKind::rlp_decode_inner_without_bloom(
            buf, tx_type,
        )?)
    }

    fn fallback_decode(buf: &mut &[u8]) -> alloy_eips::eip2718::Eip2718Result<Self> {
        Ok(ArbReceiptKind::rlp_decode_inner_without_bloom(
            buf,
            ArbTxType::ArbitrumLegacyTx,
        )?)
    }
}

impl ArbReceipt {
    /// Decode a standard EVM receipt (EIP-2930, EIP-1559, EIP-7702) from RLP.
    fn decode_standard_receipt(
        buf: &mut &[u8],
        wrap: impl FnOnce(AlloyReceipt) -> ArbReceiptKind,
    ) -> alloy_eips::eip2718::Eip2718Result<Self> {
        let header = alloy_rlp::Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString.into());
        }
        let remaining = buf.len();
        let status: Eip658Value = alloy_rlp::Decodable::decode(buf)?;
        let cumulative_gas_used: u64 = alloy_rlp::Decodable::decode(buf)?;
        let logs: Vec<Log> = alloy_rlp::Decodable::decode(buf)?;
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength.into());
        }
        Ok(ArbReceipt::new(wrap(AlloyReceipt {
            status,
            cumulative_gas_used,
            logs,
        })))
    }
}

// ---------------------------------------------------------------------------
// Encodable / Decodable (RLP) — network encoding
// ---------------------------------------------------------------------------

impl alloy_rlp::Encodable for ArbReceipt {
    fn encode(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        self.network_encode(out);
    }

    fn length(&self) -> usize {
        self.network_len()
    }
}

impl alloy_rlp::Decodable for ArbReceipt {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Ok(Self::network_decode(buf)?)
    }
}

// ---------------------------------------------------------------------------
// serde
// ---------------------------------------------------------------------------

impl serde::Serialize for ArbReceipt {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut state = serializer.serialize_struct("ArbReceipt", 5)?;
        state.serialize_field("status", &self.status())?;
        state.serialize_field("cumulative_gas_used", &self.cumulative_gas_used())?;
        state.serialize_field("ty", &self.ty())?;
        state.serialize_field("gas_used_for_l1", &self.gas_used_for_l1)?;
        state.serialize_field("multi_gas_used", &self.multi_gas_used)?;
        state.end()
    }
}

impl<'de> serde::Deserialize<'de> for ArbReceipt {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        #[derive(serde::Deserialize)]
        struct Helper {
            status: bool,
            cumulative_gas_used: u64,
            #[serde(default)]
            ty: u8,
            #[serde(default)]
            gas_used_for_l1: u64,
            #[serde(default)]
            multi_gas_used: MultiGas,
        }
        let helper = Helper::deserialize(deserializer)?;
        let kind = if helper.ty == 0x64 {
            ArbReceiptKind::Deposit(ArbDepositReceipt)
        } else {
            let receipt = AlloyReceipt {
                status: Eip658Value::Eip658(helper.status),
                cumulative_gas_used: helper.cumulative_gas_used,
                logs: Vec::new(),
            };
            ArbReceiptKind::Legacy(receipt)
        };
        Ok(ArbReceipt {
            kind,
            gas_used_for_l1: helper.gas_used_for_l1,
            multi_gas_used: helper.multi_gas_used,
        })
    }
}

// ---------------------------------------------------------------------------
// RlpBincode — required by SerdeBincodeCompat
// ---------------------------------------------------------------------------

impl reth_primitives_traits::serde_bincode_compat::RlpBincode for ArbReceipt {}

// ---------------------------------------------------------------------------
// Compact — storage encoding (includes gas_used_for_l1)
// ---------------------------------------------------------------------------

impl reth_codecs::Compact for ArbReceipt {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        // Encode the receipt body via 2718.
        let mut encoded = Vec::new();
        self.encode_2718(&mut encoded);
        let len = encoded.len() as u32;
        buf.put_u32(len);
        buf.put_slice(&encoded);
        // Append gas_used_for_l1 for storage.
        buf.put_u64(self.gas_used_for_l1);
        // Append multi_gas_used (8 dimensions + total + refund = 10 u64s).
        for i in 0..crate::multigas::NUM_RESOURCE_KIND {
            buf.put_u64(
                self.multi_gas_used.get(
                    crate::multigas::ResourceKind::from_u8(i as u8)
                        .unwrap_or(crate::multigas::ResourceKind::Unknown),
                ),
            );
        }
        buf.put_u64(self.multi_gas_used.total());
        buf.put_u64(self.multi_gas_used.refund());
        0
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        use bytes::Buf;
        let mut slice = buf;
        let receipt_len = slice.get_u32() as usize;
        let receipt_bytes = &slice[..receipt_len];
        slice = &slice[receipt_len..];

        let mut rbuf = receipt_bytes;
        let mut receipt = Self::network_decode(&mut rbuf).unwrap_or_else(|_| {
            ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
                status: Eip658Value::Eip658(false),
                cumulative_gas_used: 0,
                logs: Vec::new(),
            }))
        });

        // Bytes after the length-prefixed body, bounded by the on-disk `len` so
        // trailing data from a larger blob is left for the caller.
        let mut tail = len.saturating_sub(4 + receipt_len).min(slice.len());

        if tail >= 8 {
            receipt.gas_used_for_l1 = slice.get_u64();
            tail -= 8;
        }

        // The multi-gas section is `[gas: K x u64][total][refund]`. K was
        // NUM_RESOURCE_KIND at write time and may differ from the current value,
        // so derive it from the section length instead of assuming it — receipts
        // written under a smaller or larger K decode without over-reading.
        if tail >= 16 {
            let kinds_on_disk = (tail - 16) / 8;
            let store = kinds_on_disk.min(crate::multigas::NUM_RESOURCE_KIND);
            let mut gas = [0u64; crate::multigas::NUM_RESOURCE_KIND];
            for slot in gas.iter_mut().take(store) {
                *slot = slice.get_u64();
            }
            // Consume any kinds beyond what this build's array holds (a tail
            // written under a larger NUM_RESOURCE_KIND) to keep the cursor aligned.
            for _ in store..kinds_on_disk {
                slice.advance(8);
            }
            let total = slice.get_u64();
            let refund = slice.get_u64();
            receipt.multi_gas_used = MultiGas::from_raw(gas, total, refund);
        }

        (receipt, slice)
    }
}

// ---------------------------------------------------------------------------
// Compress / Decompress — delegates to Compact for database storage
// ---------------------------------------------------------------------------

impl reth_db_api::table::Compress for ArbReceipt {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let _ = reth_codecs::Compact::to_compact(self, buf);
    }
}

impl reth_db_api::table::Decompress for ArbReceipt {
    fn decompress(value: &[u8]) -> Result<Self, reth_db_api::DatabaseError> {
        let (obj, _) = reth_codecs::Compact::from_compact(value, value.len());
        Ok(obj)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn alloy_receipt() -> AlloyReceipt {
        AlloyReceipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: Vec::new(),
        }
    }

    #[test]
    fn as_receipt_returns_none_for_deposit() {
        let kind = ArbReceiptKind::Deposit(ArbDepositReceipt);
        assert!(kind.as_receipt().is_none());
    }

    #[test]
    fn as_receipt_returns_some_for_non_deposit() {
        let kind = ArbReceiptKind::Legacy(alloy_receipt());
        assert!(kind.as_receipt().is_some());
    }

    #[test]
    fn tx_receipt_trait_methods_handle_deposit() {
        let receipt = ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt));
        assert!(receipt.status());
        assert_eq!(receipt.status_or_post_state(), Eip658Value::Eip658(true));
        assert_eq!(receipt.bloom(), Bloom::ZERO);
        assert_eq!(receipt.cumulative_gas_used(), 0);
        assert!(receipt.logs().is_empty());
    }

    #[test]
    fn tx_receipt_trait_methods_delegate_for_non_deposit() {
        let receipt = ArbReceipt::new(ArbReceiptKind::Legacy(alloy_receipt()));
        assert!(receipt.status());
        assert_eq!(receipt.cumulative_gas_used(), 21_000);
    }

    const NUM: usize = crate::multigas::NUM_RESOURCE_KIND;

    /// Encode a receipt with an arbitrary `gas.len()` multi-gas width, mirroring
    /// `to_compact` so tests can replay tails written under a different
    /// `NUM_RESOURCE_KIND`.
    fn compact_with_kinds(receipt: &ArbReceipt, gas: &[u64], total: u64, refund: u64) -> Vec<u8> {
        use bytes::BufMut;
        let mut body = Vec::new();
        receipt.encode_2718(&mut body);
        let mut buf = Vec::new();
        buf.put_u32(body.len() as u32);
        buf.put_slice(&body);
        buf.put_u64(receipt.gas_used_for_l1);
        for &g in gas {
            buf.put_u64(g);
        }
        buf.put_u64(total);
        buf.put_u64(refund);
        buf
    }

    fn decode(buf: &[u8]) -> ArbReceipt {
        let (receipt, rest) = <ArbReceipt as reth_codecs::Compact>::from_compact(buf, buf.len());
        assert!(rest.is_empty(), "{} trailing bytes left", rest.len());
        receipt
    }

    #[test]
    fn from_compact_round_trips_current_kinds() {
        let mut receipt = ArbReceipt::new(ArbReceiptKind::Eip1559(alloy_receipt()));
        receipt.gas_used_for_l1 = 4242;
        let gas: [u64; NUM] = std::array::from_fn(|i| (i as u64 + 1) * 10);
        receipt.multi_gas_used = MultiGas::from_raw(gas, 999, 7);
        let mut buf = Vec::new();
        reth_codecs::Compact::to_compact(&receipt, &mut buf);
        let decoded = decode(&buf);
        assert_eq!(decoded.gas_used_for_l1, 4242);
        assert_eq!(decoded.multi_gas_used, receipt.multi_gas_used);
    }

    #[test]
    fn from_compact_pads_receipt_with_fewer_kinds() {
        // An 8-kind, 80-byte tail written before NUM_RESOURCE_KIND grew — the
        // case that previously over-read and panicked.
        let mut receipt = ArbReceipt::new(ArbReceiptKind::Eip1559(alloy_receipt()));
        receipt.gas_used_for_l1 = 11;
        let gas: Vec<u64> = (1..=8).map(|n| n * 100).collect();
        let decoded = decode(&compact_with_kinds(&receipt, &gas, 800, 5));
        let mut expected = [0u64; NUM];
        expected[..gas.len()].copy_from_slice(&gas);
        assert_eq!(decoded.gas_used_for_l1, 11);
        assert_eq!(decoded.multi_gas_used, MultiGas::from_raw(expected, 800, 5));
    }

    // Canonical Arbitrum Sepolia block 1 auto-redeem retry tx (0x873c5ee3…);
    // fields fetched from a real node. Proves our typed-tx hash preimage matches.
    #[test]
    fn canonical_block1_retry_tx_hash() {
        use alloy_primitives::{address, b256, hex, keccak256, Bytes, U256};
        use arb_alloy_consensus::tx::ArbRetryTx;
        let tx = ArbRetryTx {
            chain_id: U256::from(421614u64),
            nonce: 0,
            from: address!("b8787d8f23e176a5d32135d746b69886e03313be"),
            gas_fee_cap: U256::from(0x5f5e100u64),
            gas: 100_000,
            to: Some(address!("3fab184622dc19b6109349b94811493bf2a45362")),
            value: U256::from(0x2386f26fc10000u64),
            data: Bytes::new(),
            ticket_id: b256!("13cb79b086a427f3db7ebe6ec2bb90a806a3b0368ecee6020144f352e37dbdf6"),
            refund_to: address!("11155ca9bbf7be58e27f3309e629c847996b43c8"),
            max_refund: U256::from(0xb0e85efeab8u64),
            submission_fee_refund: U256::from(0x1f6377d4ab8u64),
        };
        let mut enc = Vec::new();
        enc.push(ArbTxType::ArbitrumRetryTx.as_u8());
        alloy_rlp::Encodable::encode(&tx, &mut enc);
        assert_eq!(
            keccak256(&enc),
            b256!("873c5ee3092c40336006808e249293bf5f4cb3235077a74cac9cafa7cf73cb8b"),
            "retry-tx hash mismatch; our preimage = 0x{}",
            hex::encode(&enc)
        );
    }

    // Canonical Arbitrum Sepolia block 1 SubmitRetryable (0x13cb79b0…) = the
    // ticketId that feeds the retry tx. Proves our type-0x69 hash preimage.
    #[test]
    fn canonical_block1_submit_retryable_hash() {
        use alloy_primitives::{address, b256, hex, keccak256, Bytes, U256};
        use arb_alloy_consensus::tx::ArbSubmitRetryableTx;
        let tx = ArbSubmitRetryableTx {
            chain_id: U256::from(421614u64),
            request_id: b256!("0000000000000000000000000000000000000000000000000000000000000001"),
            from: address!("b8787d8f23e176a5d32135d746b69886e03313be"),
            l1_base_fee: U256::from(0x5bd57bd9u64),
            deposit_value: U256::from(0x23e3dbb7b88ab8u64),
            gas_fee_cap: U256::from(0x3b9aca00u64),
            gas: 100_000,
            retry_to: Some(address!("3fab184622dc19b6109349b94811493bf2a45362")),
            retry_value: U256::from(0x2386f26fc10000u64),
            beneficiary: address!("11155ca9bbf7be58e27f3309e629c847996b43c8"),
            max_submission_fee: U256::from(0x1f6377d4ab8u64),
            fee_refund_addr: address!("11155ca9bbf7be58e27f3309e629c847996b43c8"),
            retry_data: Bytes::new(),
        };
        let mut enc = Vec::new();
        enc.push(ArbTxType::ArbitrumSubmitRetryableTx.as_u8());
        alloy_rlp::Encodable::encode(&tx, &mut enc);
        assert_eq!(
            keccak256(&enc),
            b256!("13cb79b086a427f3db7ebe6ec2bb90a806a3b0368ecee6020144f352e37dbdf6"),
            "submit-retryable hash mismatch; our preimage = 0x{}",
            hex::encode(&enc)
        );
    }

    #[test]
    fn from_compact_truncates_receipt_with_more_kinds() {
        // A tail written under a larger NUM_RESOURCE_KIND: extra kinds are
        // consumed (no over-read) but dropped.
        let mut receipt = ArbReceipt::new(ArbReceiptKind::Eip1559(alloy_receipt()));
        receipt.gas_used_for_l1 = 22;
        let gas: Vec<u64> = (1..=(NUM as u64 + 1)).map(|n| n * 100).collect();
        let decoded = decode(&compact_with_kinds(&receipt, &gas, 1000, 9));
        let mut expected = [0u64; NUM];
        expected.copy_from_slice(&gas[..NUM]);
        assert_eq!(
            decoded.multi_gas_used,
            MultiGas::from_raw(expected, 1000, 9)
        );
    }
}
