// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package txs

import (
	"context"
	"math/big"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type Tx struct {
	DataCreatedAt time.Time
	Nonce         uint64
	Meta          []byte
	To            common.Address
	Calldata      []byte
	GasLimit      uint64
	Value         *big.Int
	KzgBlobs      []kzg4844.Blob
	AccessList    types.AccessList
}

type dataPoster interface {
	Client() *ethclient.Client
	Config() *config.DataPosterConfig
	ExtraBacklog() uint64
	HeaderReader() *headerreader.HeaderReader
	InternalState() *state.InternalState
	MaxFeeCapExpression() *govaluate.EvaluableExpression
	ParentChain() *parent.ParentChain
	ParentChainID256() *uint256.Int
	RetrieveMetadata(context.Context, *big.Int) ([]byte, error)
	Sender() common.Address
	Signer(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error)
	UsingNoOpStorage() bool
}
