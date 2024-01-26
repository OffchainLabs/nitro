// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use num_enum::{IntoPrimitive, TryFromPrimitive};

// These values must be kept in sync with `arbutil/preimage_type.go`,
// and the if statement in `contracts/src/osp/OneStepProverHostIo.sol` (search for "UNKNOWN_PREIMAGE_TYPE").
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, TryFromPrimitive, IntoPrimitive,
)]
#[repr(u8)]
pub enum PreimageType {
    Keccak256,
    Sha2_256,
    EthVersionedHash,
}
