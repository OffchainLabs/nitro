// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sol

import "github.com/offchainlabs/nitro/bold/protocol"

var (
	_ = protocol.SpecEdge(&specEdge{})
	_ = protocol.SpecChallengeManager(&specChallengeManager{})
)
