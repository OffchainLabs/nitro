//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/util"
)

func NewTransactionStreamerForTest(t *testing.T, ownerAddress common.Address) (*TransactionStreamer, *core.BlockChain) {
	rewrittenOwnerAddress := util.RemapL1Address(ownerAddress)

	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[rewrittenOwnerAddress] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether),
		Nonce:      0,
		PrivateKey: nil,
	}
	genesis := &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumTest"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	db := rawdb.NewMemoryDatabase()
	genesis.MustCommit(db)
	shouldPreserve := func(_ *types.Block) bool { return false }
	bc, err := core.NewBlockChain(db, nil, arbos.ChainConfig, arbos.Engine{}, vm.Config{}, shouldPreserve, nil)
	if err != nil {
		t.Fatal(err)
	}

	inbox, err := NewTransactionStreamer(db, bc)
	if err != nil {
		t.Fatal(err)
	}

	return inbox, bc
}

func TestBlockGasLimit(t *testing.T) {
	ownerAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")

	inbox, bc := NewTransactionStreamerForTest(t, ownerAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inbox.Start(ctx)

	_ = bc
}
