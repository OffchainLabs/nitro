//! Value codecs for the consensus DB: RLP field helpers ([`rlp`]), the legacy L1
//! wire format ([`legacy`]), and small shared helpers.

pub mod legacy;
pub mod rlp;

use alloy_primitives::B256;

use crate::{ConsensusDbError, Result};

/// Split a stored value into its leading 32-byte accumulator and the remaining
/// payload bytes. Both delayed-message prefixes (`d`/`e`) store `accumulator ++ message`.
pub fn strip_accumulator(bytes: &[u8]) -> Result<(B256, &[u8])> {
    let accumulator = B256::from_slice(
        bytes
            .get(..32)
            .ok_or(ConsensusDbError::InvalidStoredValue)?,
    );
    Ok((accumulator, &bytes[32..]))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn splits_accumulator_from_payload() {
        let mut bytes = vec![0xAA; 32];
        bytes.extend_from_slice(&[1, 2, 3]);
        let (acc, rest) = strip_accumulator(&bytes).unwrap();
        assert_eq!(acc, B256::repeat_byte(0xAA));
        assert_eq!(rest, &[1, 2, 3]);
    }

    #[test]
    fn empty_payload_is_allowed() {
        let bytes = [0xBB; 32];
        let (acc, rest) = strip_accumulator(&bytes).unwrap();
        assert_eq!(acc, B256::repeat_byte(0xBB));
        assert!(rest.is_empty());
    }

    #[test]
    fn short_input_errors() {
        let bytes = [0u8; 31];
        assert!(matches!(
            strip_accumulator(&bytes),
            Err(ConsensusDbError::InvalidStoredValue)
        ));
    }
}
