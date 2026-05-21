// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mainloop

import (
	"context"
	"math/big"

	"github.com/Knetic/govaluate"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type dataPoster interface {
	Client() *ethclient.Client
	Config() *config.DataPosterConfig
	ExtraBacklog() uint64
	HeaderReader() *headerreader.HeaderReader
	InternalState() *state.InternalState
	MaxFeeCapExpression() *govaluate.EvaluableExpression
	ParentChain() *parent.ParentChain
	RetrieveMetadata(context.Context, *big.Int) ([]byte, error)
	Sender() common.Address
	Signer(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error)
	UsingNoOpStorage() bool
}
