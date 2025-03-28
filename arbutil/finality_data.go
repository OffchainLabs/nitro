// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import "github.com/ethereum/go-ethereum/common"

type FinalityData struct {
	MsgIdx    MessageIndex
	BlockHash common.Hash
}
