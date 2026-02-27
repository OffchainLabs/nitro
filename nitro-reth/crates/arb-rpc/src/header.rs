//! Arbitrum header conversion for RPC responses.
//!
//! Extracts Arbitrum-specific fields (sendRoot, sendCount, l1BlockNumber)
//! from the consensus header's mix_hash and extra_data fields.

use alloy_consensus::{BlockHeader, Header};
use alloy_primitives::{B256, U256};
use alloy_rpc_types_eth::Header as RpcHeader;
use alloy_serde::WithOtherFields;
use reth_primitives_traits::SealedHeader;
use reth_rpc_convert::transaction::HeaderConverter;
use std::convert::Infallible;

/// Converts consensus headers to RPC headers with Arbitrum extension fields.
#[derive(Debug, Clone)]
pub struct ArbHeaderConverter;

impl HeaderConverter<Header, WithOtherFields<RpcHeader<Header>>> for ArbHeaderConverter {
    type Err = Infallible;

    fn convert_header(
        &self,
        header: SealedHeader<Header>,
        block_size: usize,
    ) -> Result<WithOtherFields<RpcHeader<Header>>, Self::Err> {
        let mix = header.mix_hash().unwrap_or_default();
        let extra = header.extra_data();

        // Extract Arbitrum fields from mix_hash.
        let send_count = u64::from_be_bytes(mix.0[0..8].try_into().unwrap_or_default());
        let l1_block_number = u64::from_be_bytes(mix.0[8..16].try_into().unwrap_or_default());

        // Send root is stored in the first 32 bytes of extra_data.
        let send_root = if extra.len() >= 32 {
            B256::from_slice(&extra[..32])
        } else {
            B256::ZERO
        };

        let base_header =
            RpcHeader::from_consensus(header.into(), None, Some(U256::from(block_size)));

        let mut other = std::collections::BTreeMap::new();
        other.insert(
            "sendRoot".to_string(),
            serde_json::to_value(send_root).unwrap_or_default(),
        );
        other.insert(
            "sendCount".to_string(),
            serde_json::to_value(format!("{send_count:#x}")).unwrap_or_default(),
        );
        other.insert(
            "l1BlockNumber".to_string(),
            serde_json::to_value(format!("{l1_block_number:#x}")).unwrap_or_default(),
        );

        Ok(WithOtherFields {
            inner: base_header,
            other: alloy_serde::OtherFields::new(other),
        })
    }
}
