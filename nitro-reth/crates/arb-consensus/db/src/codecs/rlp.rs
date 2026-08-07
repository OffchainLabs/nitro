use alloy_rlp::{Decodable, Encodable};

use crate::{Result, schema::ConsensusDbValue};

/// A generic RLP-encoded value wrapper. Implements [`ConsensusDbValue`] for any
/// `T: Encodable + Decodable`.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Rlp<T>(pub T);

impl<T: Encodable + Decodable> ConsensusDbValue for Rlp<T> {
    fn encode(&self) -> Vec<u8> {
        alloy_rlp::encode(&self.0)
    }

    fn decode(bytes: &[u8]) -> Result<Self> {
        Ok(Rlp(alloy_rlp::decode_exact(bytes)?))
    }
}
