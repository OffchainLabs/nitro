//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"
	"encoding/binary"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbos/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

type blockTestState struct {
	balances    map[common.Address]uint64
	accounts    []common.Address
	numMessages uint64
	blockNumber uint64
}

func TestTransactionStreamer(t *testing.T) {
	ownerAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inbox.Start(ctx)

	var blockStates []blockTestState
	blockStates = append(blockStates, blockTestState{
		balances: map[common.Address]uint64{
			rewrittenOwnerAddress: params.Ether,
		},
		accounts:    []common.Address{rewrittenOwnerAddress},
		numMessages: 0,
		blockNumber: 0,
	})
	for i := 1; i < 100; i++ {
		if i%10 == 0 {
			reorgTo := rand.Int() % len(blockStates)
			err = inbox.ReorgTo(blockStates[reorgTo].numMessages)
			if err != nil {
				t.Fatal(err)
			}
			blockStates = blockStates[:(reorgTo + 1)]
		} else {
			state := blockStates[len(blockStates)-1]
			newBalances := make(map[common.Address]uint64)
			for k, v := range state.balances {
				newBalances[k] = v
			}
			state.balances = newBalances

			var messages []arbstate.MessageWithMetadata
			// TODO replay a random amount of messages too
			numMessages := rand.Int() % 5
			for j := 0; j < numMessages; j++ {
				source := state.accounts[rand.Int()%len(state.accounts)]
				amount := uint64(rand.Int()) % state.balances[source]
				if state.balances[source]-amount < 100000000 {
					// Leave enough funds for gas
					amount = 1
				}
				var dest common.Address
				if j == 0 {
					binary.LittleEndian.PutUint64(dest[:], uint64(len(state.accounts)))
					state.accounts = append(state.accounts, dest)
				} else {
					dest = state.accounts[rand.Int()%len(state.accounts)]
				}
				var gas uint64 = 100000
				var l2Message []byte
				l2Message = append(l2Message, arbos.L2MessageKind_ContractTx)
				l2Message = append(l2Message, math.U256Bytes(new(big.Int).SetUint64(gas))...)
				l2Message = append(l2Message, math.U256Bytes(big.NewInt(1))...)
				l2Message = append(l2Message, dest.Hash().Bytes()...)
				l2Message = append(l2Message, math.U256Bytes(new(big.Int).SetUint64(amount))...)
				messages = append(messages, arbstate.MessageWithMetadata{
					Message: &arbos.L1IncomingMessage{
						Header: &arbos.L1IncomingMessageHeader{
							Kind:   arbos.L1MessageType_L2Message,
							Sender: util.InverseRemapL1Address(source),
						},
						L2msg: l2Message,
					},
					MustEndBlock:        true,
					DelayedMessagesRead: 0,
				})
				state.balances[source] -= amount
				state.balances[dest] += amount
			}

			err = inbox.AddMessages(state.numMessages, false, messages)
			if err != nil {
				t.Fatal(err)
			}

			state.numMessages += uint64(len(messages))
			state.blockNumber += uint64(len(messages))
			for i := 0; ; i++ {
				blockNumber := bc.CurrentHeader().Number.Uint64()
				if blockNumber > state.blockNumber {
					t.Fatal("unexpected block number", blockNumber, ">", state.blockNumber)
				} else if blockNumber == state.blockNumber {
					break
				} else if i >= 100 {
					t.Fatal("timed out waiting for new block")
				}
				time.Sleep(10 * time.Millisecond)
			}
			blockStates = append(blockStates, state)
		}

		// Check that state balances are consistent with blockchain's balances
		lastBlockNumber := bc.CurrentHeader().Number.Uint64()
		expectedLastBlockNumber := blockStates[len(blockStates)-1].blockNumber
		if lastBlockNumber != expectedLastBlockNumber {
			t.Fatal("unexpected block number", lastBlockNumber, "vs", expectedLastBlockNumber)
		}

		for _, state := range blockStates {
			block := bc.GetBlockByNumber(state.blockNumber)
			if block == nil {
				t.Fatal("missing state block", state.blockNumber)
			}
			for acct, balance := range state.balances {
				state, err := bc.StateAt(block.Root())
				if err != nil {
					t.Fatal("error getting block state", err)
				}
				haveBalance := state.GetBalance(acct)
				if new(big.Int).SetUint64(balance).Cmp(haveBalance) < 0 {
					t.Error("unexpected balance for account", acct, "; expected", balance, "got", haveBalance)
				}
			}
		}
	}
}
