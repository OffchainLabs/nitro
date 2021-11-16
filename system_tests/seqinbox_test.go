//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

type blockTestState struct {
	balances      map[common.Address]uint64
	nonces        map[common.Address]uint64
	accounts      []common.Address
	l2BlockNumber uint64
	l1BlockNumber uint64
}

func TestSequencerInboxReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Info, arbNode, l1Info, l1backend, stack := CreateTestNodeOnL1(t, ctx, true)
	l2Backend := arbNode.Backend
	defer stack.Close()
	l1Client := l1Info.Client

	l1BlockChain := l1backend.BlockChain()

	seqInbox, err := bridgegen.NewSequencerInbox(l1Info.GetAddress("SequencerInbox"), l1Client)
	if err != nil {
		t.Fatal(err)
	}
	seqOpts := l1Info.GetDefaultTransactOpts("Sequencer")

	ownerAddress := l2Info.GetAddress("Owner")
	startL2BlockNumber := l2Backend.APIBackend().CurrentHeader().Number.Uint64()

	startState, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		t.Fatal(err)
	}
	startOwnerBalance := startState.GetBalance(ownerAddress).Uint64()
	startOwnerNonce := startState.GetNonce(ownerAddress)

	var blockStates []blockTestState
	blockStates = append(blockStates, blockTestState{
		balances: map[common.Address]uint64{
			ownerAddress: startOwnerBalance,
		},
		nonces: map[common.Address]uint64{
			ownerAddress: startOwnerNonce,
		},
		accounts:      []common.Address{ownerAddress},
		l2BlockNumber: startL2BlockNumber,
	})

	accountName := func(x int) string {
		if x == 0 {
			return "Owner"
		} else {
			return "Account" + strconv.Itoa(x)
		}
	}

	l1Info.GenerateAccount("ReorgPadding")
	SendWaitTestTransactions(t, ctx, l1Client, []*types.Transaction{
		l1Info.PrepareTx("faucet", "ReorgPadding", 30000, big.NewInt(1e14), nil),
	})

	for i := 1; i < 40; i++ {
		if i%10 == 0 {
			reorgTo := rand.Int() % len(blockStates)
			if reorgTo == 0 {
				reorgTo = 1
			}
			// Make the reorg larger to force the miner to discard transactions.
			// The miner usually collects transactions from deleted blocks and puts them in the mempool.
			// However, this code doesn't run on reorgs larger than 64 blocks for performance reasons.
			// Therefore, we make a bunch of small blocks to prevent the code from running.
			padAddr := l1Info.GetAddress("ReorgPadding")
			for j := uint64(0); j < 65; j++ {
				rawTx := &types.DynamicFeeTx{
					To:        &padAddr,
					Gas:       21000,
					GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
					Value:     new(big.Int),
					Nonce:     j,
				}
				tx := l1Info.SignTxAs("ReorgPadding", rawTx)
				SendWaitTestTransactions(t, ctx, l1Client, []*types.Transaction{tx})
			}
			reorgTargetNumber := blockStates[reorgTo].l1BlockNumber
			currentHeader, err := l1Client.HeaderByNumber(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if currentHeader.Number.Int64()-int64(reorgTargetNumber) < 65 {
				t.Fatal("Less than 65 blocks of difference between current block", currentHeader.Number, "and target", reorgTargetNumber)
			}
			t.Logf("Reorganizing to L1 block %v", reorgTargetNumber)
			reorgTarget := l1BlockChain.GetBlockByNumber(reorgTargetNumber)
			err = l1BlockChain.ReorgToOldBlock(reorgTarget)
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
			newNonces := make(map[common.Address]uint64)
			for k, v := range state.nonces {
				newNonces[k] = v
			}
			state.nonces = newNonces

			batchBuffer := bytes.NewBuffer([]byte{0})
			batchWriter := brotli.NewWriter(batchBuffer)
			numMessages := 1 + rand.Int()%5
			for j := 0; j < numMessages; j++ {
				sourceNum := rand.Int() % len(state.accounts)
				source := state.accounts[sourceNum]
				amount := uint64(rand.Int()) % state.balances[source]
				if state.balances[source]-amount < params.InitialBaseFee*10000000 {
					// Leave enough funds for gas
					amount = 1
				}
				var dest common.Address
				if j == 0 && amount >= params.InitialBaseFee*1000000 {
					name := accountName(len(state.accounts))
					l2Info.GenerateAccount(name)
					dest = l2Info.GetAddress(name)
					state.accounts = append(state.accounts, dest)
				} else {
					dest = state.accounts[rand.Int()%len(state.accounts)]
				}

				rawTx := &types.DynamicFeeTx{
					To:        &dest,
					Gas:       21000,
					GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
					Value:     new(big.Int).SetUint64(amount),
					Nonce:     state.nonces[source],
				}
				state.nonces[source]++
				tx := l2Info.SignTxAs(accountName(sourceNum), rawTx)
				txData, err := tx.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var segment []byte
				segment = append(segment, arbstate.BatchSegmentKindL2Message)
				segment = append(segment, arbos.L2MessageKind_SignedTx)
				segment = append(segment, txData...)
				err = rlp.Encode(batchWriter, segment)
				if err != nil {
					t.Fatal(err)
				}

				state.balances[source] -= amount
				state.balances[dest] += amount
			}

			err = batchWriter.Close()
			if err != nil {
				t.Fatal(err)
			}
			batchData := batchBuffer.Bytes()

			var tx *types.Transaction
			if i%5 == 0 {
				tx, err = seqInbox.AddSequencerL2Batch(&seqOpts, big.NewInt(int64(len(blockStates)-1)), batchData, big.NewInt(0), common.Address{})
			} else {
				tx, err = seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(int64(len(blockStates)-1)), batchData, big.NewInt(0), common.Address{})
			}
			if err != nil {
				t.Fatal(err)
			}
			txRes, err := arbnode.EnsureTxSucceeded(ctx, l1Client, tx)
			if err != nil {
				t.Fatal(err)
			}

			state.l2BlockNumber += uint64(numMessages)
			state.l1BlockNumber = txRes.BlockNumber.Uint64()
			blockStates = append(blockStates, state)
		}

		t.Logf("Iteration %v: state %v block %v", i, len(blockStates)-1, blockStates[len(blockStates)-1].l2BlockNumber)

		for i := 0; ; i++ {
			batchCount, err := seqInbox.BatchCount(&bind.CallOpts{})
			if err != nil {
				t.Fatal(err)
			}
			if batchCount.Cmp(big.NewInt(int64(len(blockStates)-1))) == 0 {
				break
			} else if i >= 100 {
				t.Fatal("timed out waiting for l1 batch count update; have", batchCount, "want", len(blockStates)-1)
			}
			time.Sleep(10 * time.Millisecond)
		}

		expectedBlockNumber := blockStates[len(blockStates)-1].l2BlockNumber
		for i := 0; ; i++ {
			blockNumber := l2Backend.APIBackend().CurrentHeader().Number.Uint64()
			if blockNumber == expectedBlockNumber {
				break
			} else if i >= 1000 {
				t.Fatal("timed out waiting for l2 block update; have", blockNumber, "want", expectedBlockNumber)
			}
			time.Sleep(10 * time.Millisecond)
		}

		for _, state := range blockStates {
			block, err := l2Backend.APIBackend().BlockByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			if err != nil {
				t.Fatal(err)
			}
			if block == nil {
				t.Fatal("missing state block", state.l2BlockNumber)
			}
			stateDb, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			if err != nil {
				t.Fatal(err)
			}
			for acct, balance := range state.balances {
				haveBalance := stateDb.GetBalance(acct)
				if new(big.Int).SetUint64(balance).Cmp(haveBalance) < 0 {
					t.Fatal("unexpected balance for account", acct, "; expected", balance, "got", haveBalance)
				}
			}
		}
	}
}
