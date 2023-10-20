// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

type PreimageType uint8

// These values must be kept in sync with `arbitrator/arbutil/src/types.rs`,
// and the if statement in `contracts/src/osp/OneStepProverHostIo.sol` (search for "UNKNOWN_PREIMAGE_TYPE").
const (
	Keccak256PreimageType PreimageType = iota
	Sha2_256PreimageType
)
