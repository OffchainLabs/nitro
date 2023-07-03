// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE
package time

var (
	_ = Reference(&realTimeReference{})
	_ = Reference(&ArtificialTimeReference{})
)
