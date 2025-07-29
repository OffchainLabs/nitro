// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/bold/blob/main/LICENSE.md

package solimpl

import protocol "github.com/offchainlabs/bold/chain-abstraction"

var (
	_ = protocol.SpecEdge(&specEdge{})
	_ = protocol.SpecChallengeManager(&specChallengeManager{})
)
