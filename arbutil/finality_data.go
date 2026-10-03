// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbutil

import "github.com/ethereum/go-ethereum/common"

// lint:require-exhaustive-initialization
type FinalityData struct {
	MsgIdx    MessageIndex
	BlockHash common.Hash
}
