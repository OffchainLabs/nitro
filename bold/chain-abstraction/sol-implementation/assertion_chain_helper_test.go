// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package solimpl

import protocol "github.com/offchainlabs/bold/chain-abstraction"

func (a *AssertionChain) SetBackend(b protocol.ChainBackend) {
	a.backend = b
}
