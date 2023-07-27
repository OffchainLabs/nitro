// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package time

var (
	_ = Reference(&realTimeReference{})
	_ = Reference(&ArtificialTimeReference{})
)
