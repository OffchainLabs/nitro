// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package fees

import (
	"math/big"

	"github.com/Knetic/govaluate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/parent"
)

// Split between blob and non-blob for calculations
type BlobSplit[T any] struct {
	Blob T
	NonBlob T
}

func (s *BlobSplit[T]) SelectIfBlobs(hasBlobs bool) T {
	if hasBlobs {
		return s.Blob
	} else {
		return s.NonBlob
	}
}

// Calculated fee and tip caps 
type Caps struct {
	Fee BlobSplit[*big.Int]
	Tip *big.Int
}

type dataPoster interface {
	Client() *ethclient.Client
	Config() *config.DataPosterConfig
	ExtraBacklog() uint64
	MaxFeeCapExpression() *govaluate.EvaluableExpression
	ParentChain() *parent.ParentChain
	Sender() common.Address
	UsingNoOpStorage() bool
}
