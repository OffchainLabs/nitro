//! Value codecs for the consensus DB: RLP field helpers ([`rlp`]), the legacy L1
//! wire format ([`legacy`]), and small shared helpers.

pub mod legacy;
pub mod rlp;

use alloy_primitives::B256;

use crate::{ConsensusDbError, Result};

/// Split a stored value into its leading 32-byte accumulator and the remaining
/// payload bytes. Both delayed-message tables (`d`/`e`) store `accumulator ++ message`.
pub fn strip_accumulator(bytes: &[u8]) -> Result<(B256, &[u8])> {
    let accumulator =
        B256::from_slice(bytes.get(..32).ok_or(ConsensusDbError::InvalidStoredValue)?);
    Ok((accumulator, &bytes[32..]))
}
