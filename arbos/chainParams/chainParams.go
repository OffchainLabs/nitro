//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package chainParams

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
)

type ChainParamID uint64

const (
	DummyParameter ChainParamID = iota
)

type ArbitrumChainParams struct {
	sto *storage.Storage
}

func InitializeArbitrumChainParams(sto *storage.Storage) {

}

func OpenArbitrumChainParams(sto *storage.Storage) *ArbitrumChainParams {
	return &ArbitrumChainParams{sto}
}

func (acp *ArbitrumChainParams) Get(which ChainParamID) common.Hash {
	return acp.sto.GetByUint64(uint64(which))
}

func (acp *ArbitrumChainParams) Set(which ChainParamID, val common.Hash) {
	acp.sto.SetByUint64(uint64(which), val)
}
