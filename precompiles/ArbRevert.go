//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

// This precompile always reverts, giving users hard errors for making calls to addresses like 0xa4b05
type ArbRevert struct {
	Address addr
}
