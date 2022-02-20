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

	"github.com/offchainlabs/arbstate/arbos/l2pricing"
	"github.com/offchainlabs/arbstate/statetransfer"

	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/util/testhelpers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

func NewTransactionStreamerForTest(t *testing.T, ownerAddress common.Address) (*TransactionStreamer, *core.BlockChain) {
	rewrittenOwnerAddress := util.RemapL1Address(ownerAddress)

	chainConfig := params.ArbitrumTestChainConfig()

	initData := statetransfer.ArbosInitializationInfo{
		Accounts: []statetransfer.AccountInitializationInfo{
			{
				Addr:       rewrittenOwnerAddress,
				EthBalance: big.NewInt(params.Ether),
			},
		},
	}

	db := rawdb.NewMemoryDatabase()
	initReader := statetransfer.NewMemoryInitDataReader(&initData)

	bc, err := WriteOrTestBlockChain(db, nil, initReader, 0, chainConfig)

	if err != nil {
		Fail(t, err)
	}

	inbox, err := NewTransactionStreamer(db, bc, nil)
	if err != nil {
		Fail(t, err)
	}

	return inbox, bc
}

type blockTestState struct {
	balances    map[common.Address]*big.Int
	accounts    []common.Address
	numMessages uint64
	blockNumber uint64
}

func TestTransactionStreamer(t *testing.T) {
	ownerAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")
	rewrittenOwnerAddress := util.RemapL1Address(ownerAddress)

	inbox, bc := NewTransactionStreamerForTest(t, ownerAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inbox.Start(ctx)

	maxExpectedGasCost := big.NewInt(l2pricing.InitialBaseFeeWei)
	maxExpectedGasCost.Mul(maxExpectedGasCost, big.NewInt(2100*2))

	minBalance := new(big.Int).Mul(maxExpectedGasCost, big.NewInt(100))

	var blockStates []blockTestState
	blockStates = append(blockStates, blockTestState{
		balances: map[common.Address]*big.Int{
			rewrittenOwnerAddress: new(big.Int).Mul(maxExpectedGasCost, big.NewInt(1_000_000)),
		},
		accounts:    []common.Address{rewrittenOwnerAddress},
		numMessages: 0,
		blockNumber: 0,
	})
	for i := 1; i < 100; i++ {
		if i%10 == 0 {
			reorgTo := rand.Int() % len(blockStates)
			err := inbox.ReorgTo(blockStates[reorgTo].numMessages)
			if err != nil {
				Fail(t, err)
			}
			blockStates = blockStates[:(reorgTo + 1)]
		} else {
			state := blockStates[len(blockStates)-1]
			newBalances := make(map[common.Address]*big.Int)
			for k, v := range state.balances {
				newBalances[k] = new(big.Int).Set(v)
			}
			state.balances = newBalances

			var messages []arbstate.MessageWithMetadata
			// TODO replay a random amount of messages too
			numMessages := rand.Int() % 5
			for j := 0; j < numMessages; j++ {
				source := state.accounts[rand.Int()%len(state.accounts)]
				if state.balances[source].Cmp(minBalance) < 0 {
					continue
				}
				amount := big.NewInt(int64(rand.Int() % 1000))
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
				l2Message = append(l2Message, math.U256Bytes(big.NewInt(l2pricing.InitialBaseFeeWei))...)
				l2Message = append(l2Message, dest.Hash().Bytes()...)
				l2Message = append(l2Message, math.U256Bytes(amount)...)
				messages = append(messages, arbstate.MessageWithMetadata{
					Message: &arbos.L1IncomingMessage{
						Header: &arbos.L1IncomingMessageHeader{
							Kind:   arbos.L1MessageType_L2Message,
							Poster: util.InverseRemapL1Address(source),
						},
						L2msg: l2Message,
					},
					DelayedMessagesRead: 0,
				})
				state.balances[source].Sub(state.balances[source], amount)
				if state.balances[dest] == nil {
					state.balances[dest] = new(big.Int)
				}
				state.balances[dest].Add(state.balances[dest], amount)
			}

			Require(t, inbox.AddMessages(state.numMessages, false, messages))

			state.numMessages += uint64(len(messages))
			state.blockNumber += uint64(len(messages))
			for i := 0; ; i++ {
				blockNumber := bc.CurrentHeader().Number.Uint64()
				if blockNumber > state.blockNumber {
					Fail(t, "unexpected block number", blockNumber, ">", state.blockNumber)
				} else if blockNumber == state.blockNumber {
					break
				} else if i >= 100 {
					Fail(t, "timed out waiting for new block")
				}
				time.Sleep(10 * time.Millisecond)
			}
			blockStates = append(blockStates, state)
		}

		// Check that state balances are consistent with blockchain's balances
		lastBlockNumber := bc.CurrentHeader().Number.Uint64()
		expectedLastBlockNumber := blockStates[len(blockStates)-1].blockNumber
		if lastBlockNumber != expectedLastBlockNumber {
			Fail(t, "unexpected block number", lastBlockNumber, "vs", expectedLastBlockNumber)
		}

		for _, state := range blockStates {
			block := bc.GetBlockByNumber(state.blockNumber)
			if block == nil {
				Fail(t, "missing state block", state.blockNumber)
			}
			for acct, balance := range state.balances {
				state, err := bc.StateAt(block.Root())
				if err != nil {
					Fail(t, "error getting block state", err)
				}
				haveBalance := state.GetBalance(acct)
				if balance.Cmp(haveBalance) < 0 {
					t.Error("unexpected balance for account", acct, "; expected", balance, "got", haveBalance)
				}
			}
		}
	}
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
