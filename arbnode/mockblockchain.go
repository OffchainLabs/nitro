// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"time"
)

type mockBlockChain struct {
	config *params.ChainConfig
}

func (bc *mockBlockChain) GetCanonicalHash(u uint64) common.Hash {
	return common.Hash{}
}

func (bc *mockBlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return nil
}

func newMockBlockChain() *mockBlockChain {
	return &mockBlockChain{
		config: &params.ChainConfig{},
	}
}

func (bc *mockBlockChain) Config() *params.ChainConfig {
	return bc.config
}

func (bc *mockBlockChain) CurrentBlock() *types.Block {
	return nil
}

func (bc *mockBlockChain) CurrentHeader() *types.Header {
	return nil
}

func (bc *mockBlockChain) Engine() consensus.Engine {
	return nil
}

func (bc *mockBlockChain) GetBlockByNumber(uint64) *types.Block {
	return nil
}

func (bc *mockBlockChain) GetHeader(common.Hash, uint64) *types.Header {
	return nil
}

func (bc *mockBlockChain) RecoverState(*types.Block) error {
	return nil
}

func (bc *mockBlockChain) ReorgToOldBlock(*types.Block) error {
	return nil
}

func (bc *mockBlockChain) Genesis() *types.Block {
	return nil
}

func (bc *mockBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return nil, nil
}

func (bc *mockBlockChain) Stop() {}

func (bc *mockBlockChain) WriteBlockAndSetHeadWithTime(
	*types.Block,
	[]*types.Receipt,
	[]*types.Log,
	*state.StateDB,
	bool,
	time.Duration,
) (core.WriteStatus, error) {
	return core.NonStatTy, nil
}
