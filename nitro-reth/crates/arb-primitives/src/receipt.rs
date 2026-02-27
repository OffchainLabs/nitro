use alloc::vec::Vec;

use alloy_consensus::{
    Eip2718EncodableReceipt, Eip658Value, Receipt as AlloyReceipt, Typed2718, TxReceipt,
};
use alloy_eips::{Decodable2718, Encodable2718};
use alloy_primitives::{Bloom, Log};
use alloy_rlp::{Decodable, Encodable};
use arb_alloy_consensus::tx::ArbTxType;
use reth_primitives_traits::InMemorySize;

/// Arbitrum receipt with per-type variants matching the transaction type.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ArbReceipt {
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

impl InMemorySize for ArbReceipt {
    fn size(&self) -> usize {
        0
    }
}

impl ArbReceipt {
    pub const fn arb_tx_type(&self) -> ArbTxType {
        match self {
            ArbReceipt::Legacy(_)
            | ArbReceipt::Eip2930(_)
            | ArbReceipt::Eip1559(_)
            | ArbReceipt::Eip7702(_) => ArbTxType::ArbitrumLegacyTx,
            ArbReceipt::Deposit(_) => ArbTxType::ArbitrumDepositTx,
            ArbReceipt::Unsigned(_) => ArbTxType::ArbitrumUnsignedTx,
            ArbReceipt::Contract(_) => ArbTxType::ArbitrumContractTx,
            ArbReceipt::Retry(_) => ArbTxType::ArbitrumRetryTx,
            ArbReceipt::SubmitRetryable(_) => ArbTxType::ArbitrumSubmitRetryableTx,
            ArbReceipt::Internal(_) => ArbTxType::ArbitrumInternalTx,
        }
    }

    /// Returns the inner AlloyReceipt (panics for Deposit).
    pub const fn as_receipt(&self) -> &AlloyReceipt {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r,
            ArbReceipt::Deposit(_) => {
                unreachable!()
            }
        }
    }

    fn rlp_encoded_fields_length(&self, bloom: &Bloom) -> usize {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.rlp_encoded_fields_length_with_bloom(bloom),
            ArbReceipt::Deposit(_) => {
                Eip658Value::Eip658(true).length()
                    + 0u64.length()
                    + bloom.length()
                    + Vec::<Log>::new().length()
            }
        }
    }

    fn rlp_encode_fields(&self, bloom: &Bloom, out: &mut dyn alloy_rlp::bytes::BufMut) {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.rlp_encode_fields_with_bloom(bloom, out),
            ArbReceipt::Deposit(_) => {
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
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => {
                r.status.encode(out);
                r.cumulative_gas_used.encode(out);
                r.logs.encode(out);
            }
            ArbReceipt::Deposit(_) => {
                Eip658Value::Eip658(true).encode(out);
                (0u64).encode(out);
                let logs: Vec<Log> = Vec::new();
                logs.encode(out);
            }
        }
    }

    fn rlp_encoded_fields_length_without_bloom(&self) -> usize {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.status.length() + r.cumulative_gas_used.length() + r.logs.length(),
            ArbReceipt::Deposit(_) => {
                Eip658Value::Eip658(true).length()
                    + (0u64).length()
                    + Vec::<Log>::new().length()
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
    ) -> alloy_rlp::Result<alloy_consensus::ReceiptWithBloom<Self>> {
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
                    receipt: ArbReceipt::Deposit(ArbDepositReceipt),
                    logs_bloom,
                })
            }
            _ => {
                let alloy_consensus::ReceiptWithBloom { receipt, logs_bloom } =
                    <AlloyReceipt as alloy_consensus::RlpDecodableReceipt>::rlp_decode_with_bloom(
                        buf,
                    )?;
                Ok(alloy_consensus::ReceiptWithBloom {
                    receipt: ArbReceipt::Legacy(receipt),
                    logs_bloom,
                })
            }
        }
    }

    fn rlp_decode_inner_without_bloom(
        buf: &mut &[u8],
        tx_type: ArbTxType,
    ) -> alloy_rlp::Result<Self> {
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
        match tx_type {
            ArbTxType::ArbitrumDepositTx => Ok(Self::Deposit(ArbDepositReceipt)),
            ArbTxType::ArbitrumUnsignedTx => Ok(Self::Unsigned(receipt)),
            ArbTxType::ArbitrumContractTx => Ok(Self::Contract(receipt)),
            ArbTxType::ArbitrumRetryTx => Ok(Self::Retry(receipt)),
            ArbTxType::ArbitrumSubmitRetryableTx => Ok(Self::SubmitRetryable(receipt)),
            ArbTxType::ArbitrumInternalTx => Ok(Self::Internal(receipt)),
            ArbTxType::ArbitrumLegacyTx => Ok(Self::Legacy(receipt)),
        }
    }
}

// ---------------------------------------------------------------------------
// TxReceipt
// ---------------------------------------------------------------------------

impl TxReceipt for ArbReceipt {
    type Log = Log;

    fn status_or_post_state(&self) -> Eip658Value {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.status_or_post_state(),
            ArbReceipt::Deposit(_) => Eip658Value::Eip658(true),
        }
    }

    fn status(&self) -> bool {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.status(),
            ArbReceipt::Deposit(_) => true,
        }
    }

    fn bloom(&self) -> Bloom {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.bloom(),
            ArbReceipt::Deposit(_) => Bloom::ZERO,
        }
    }

    fn cumulative_gas_used(&self) -> u64 {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.cumulative_gas_used(),
            ArbReceipt::Deposit(_) => 0,
        }
    }

    fn logs(&self) -> &[Self::Log] {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.logs(),
            ArbReceipt::Deposit(_) => &[],
        }
    }

    fn into_logs(self) -> Vec<Self::Log> {
        match self {
            ArbReceipt::Legacy(r)
            | ArbReceipt::Eip2930(r)
            | ArbReceipt::Eip1559(r)
            | ArbReceipt::Eip7702(r)
            | ArbReceipt::Unsigned(r)
            | ArbReceipt::Contract(r)
            | ArbReceipt::Retry(r)
            | ArbReceipt::SubmitRetryable(r)
            | ArbReceipt::Internal(r) => r.logs,
            ArbReceipt::Deposit(_) => Vec::new(),
        }
    }
}

// ---------------------------------------------------------------------------
// Typed2718
// ---------------------------------------------------------------------------

impl alloy_consensus::Typed2718 for ArbReceipt {
    fn is_legacy(&self) -> bool {
        matches!(self, ArbReceipt::Legacy(_))
    }

    fn ty(&self) -> u8 {
        match self {
            ArbReceipt::Legacy(_) => 0x00,
            ArbReceipt::Eip2930(_) => 0x01,
            ArbReceipt::Eip1559(_) => 0x02,
            ArbReceipt::Eip7702(_) => 0x04,
            ArbReceipt::Deposit(_) => 0x64,
            ArbReceipt::Unsigned(_) => 0x65,
            ArbReceipt::Contract(_) => 0x66,
            ArbReceipt::Retry(_) => 0x68,
            ArbReceipt::SubmitRetryable(_) => 0x69,
            ArbReceipt::Internal(_) => 0x6A,
        }
    }
}

// ---------------------------------------------------------------------------
// Eip2718EncodableReceipt
// ---------------------------------------------------------------------------

impl alloy_consensus::Eip2718EncodableReceipt for ArbReceipt {
    fn eip2718_encoded_length_with_bloom(&self, bloom: &Bloom) -> usize {
        let inner_len = self.rlp_header_inner(bloom).length_with_payload();
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
        self.rlp_header_inner(bloom).encode(out);
        self.rlp_encode_fields(bloom, out);
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
            return ArbReceipt::rlp_decode_inner(buf, ArbTxType::ArbitrumLegacyTx);
        }
        *buf = *header_buf;
        let remaining = buf.len();
        let ty = u8::decode(buf)?;
        let tx_type = ArbTxType::from_u8(ty)
            .map_err(|_| alloy_rlp::Error::Custom("unexpected arb receipt tx type"))?;
        let this = ArbReceipt::rlp_decode_inner(buf, tx_type)?;
        if buf.len() + header.payload_length != remaining {
            return Err(alloy_rlp::Error::UnexpectedLength);
        }
        Ok(this)
    }
}

// ---------------------------------------------------------------------------
// Encodable2718 / Decodable2718
// ---------------------------------------------------------------------------

impl alloy_eips::Encodable2718 for ArbReceipt {
    fn encode_2718_len(&self) -> usize {
        let type_len = if self.is_legacy() { 0 } else { 1 };
        type_len + self.rlp_header_inner_without_bloom().length_with_payload()
    }

    fn encode_2718(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        if !self.is_legacy() {
            out.put_u8(self.ty());
        }
        self.rlp_header_inner_without_bloom().encode(out);
        self.rlp_encode_fields_without_bloom(out);
    }
}

impl alloy_eips::Decodable2718 for ArbReceipt {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> alloy_eips::eip2718::Eip2718Result<Self> {
        let tx_type = ArbTxType::from_u8(ty)
            .map_err(|_| alloy_eips::eip2718::Eip2718Error::UnexpectedType(ty))?;
        Ok(Self::rlp_decode_inner_without_bloom(buf, tx_type)?)
    }

    fn fallback_decode(buf: &mut &[u8]) -> alloy_eips::eip2718::Eip2718Result<Self> {
        Ok(Self::rlp_decode_inner_without_bloom(buf, ArbTxType::ArbitrumLegacyTx)?)
    }
}

// ---------------------------------------------------------------------------
// Encodable / Decodable (RLP)
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
        let mut state = serializer.serialize_struct("ArbReceipt", 3)?;
        state.serialize_field("status", &self.status())?;
        state.serialize_field("cumulative_gas_used", &self.cumulative_gas_used())?;
        state.serialize_field("ty", &self.ty())?;
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
        }
        let helper = Helper::deserialize(deserializer)?;
        if helper.ty == 0x64 {
            return Ok(ArbReceipt::Deposit(ArbDepositReceipt));
        }
        let receipt = AlloyReceipt {
            status: alloy_consensus::Eip658Value::Eip658(helper.status),
            cumulative_gas_used: helper.cumulative_gas_used,
            logs: Vec::new(),
        };
        Ok(ArbReceipt::Legacy(receipt))
    }
}

// ---------------------------------------------------------------------------
// RlpBincode — required by SerdeBincodeCompat
// ---------------------------------------------------------------------------

impl reth_primitives_traits::serde_bincode_compat::RlpBincode for ArbReceipt {}

// ---------------------------------------------------------------------------
// Compact
// ---------------------------------------------------------------------------

impl reth_codecs::Compact for ArbReceipt {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        let mut encoded = Vec::new();
        self.encode_2718(&mut encoded);
        let len = encoded.len() as u32;
        buf.put_u32(len);
        buf.put_slice(&encoded);
        0
    }

    fn from_compact(buf: &[u8], _len: usize) -> (Self, &[u8]) {
        use bytes::Buf;
        let mut slice = buf;
        let receipt_len = slice.get_u32() as usize;
        let receipt_bytes = &slice[..receipt_len];
        slice = &slice[receipt_len..];

        let mut rbuf = receipt_bytes;
        let receipt = Self::network_decode(&mut rbuf).unwrap_or_else(|_| {
            ArbReceipt::Legacy(AlloyReceipt {
                status: alloy_consensus::Eip658Value::Eip658(false),
                cumulative_gas_used: 0,
                logs: Vec::new(),
            })
        });

        (receipt, slice)
    }
}
