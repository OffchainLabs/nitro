// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package solimplementation

import "github.com/offchainlabs/nitro/bold/chainabstraction"

var (
	_ = chainabstraction.SpecEdge(&specEdge{})
	_ = chainabstraction.SpecChallengeManager(&specChallengeManager{})
)
