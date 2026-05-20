// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package lifecycle

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type NonceAndMeta struct {
	Nonce uint64
	Meta  []byte
}

type NonceAndMaybeMeta struct {
	Nonce            uint64
	Meta             []byte
	MetaPresent      bool
	CumulativeWeight uint64
}

type dataPoster interface {
	Client() *ethclient.Client
	Config() *config.DataPosterConfig
	HeaderReader() *headerreader.HeaderReader
	InternalState() *state.InternalState
	ParentChain() *parent.ParentChain
	RetrieveMetadata(context.Context, *big.Int) ([]byte, error)
	Sender() common.Address
}
