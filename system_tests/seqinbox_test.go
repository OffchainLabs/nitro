//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"math/rand"
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

const seqInboxTestIters = 40

func TestSequencerInboxReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Info, arbNode, l1Info, l1backend, stack := CreateTestNodeOnL1(t, ctx, false)
	l2Backend := arbNode.Backend
	defer stack.Close()
	l1Client := l1Info.Client

	l1BlockChain := l1backend.BlockChain()

	seqInbox, err := bridgegen.NewSequencerInbox(l1Info.GetAddress("SequencerInbox"), l1Client)
	Require(t, err)
	seqOpts := l1Info.GetDefaultTransactOpts("Sequencer")

	ownerAddress := l2Info.GetAddress("Owner")
	startL2BlockNumber := l2Backend.APIBackend().CurrentHeader().Number.Uint64()

	startState, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	Require(t, err)
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
			return fmt.Sprintf("Account%v", x)
		}
	}

	accounts := []string{"ReorgPadding"}
	for i := 1; i <= (seqInboxTestIters-1)/10; i++ {
		accounts = append(accounts, fmt.Sprintf("ReorgSacrifice%v", i))
	}
	var faucetTxs []*types.Transaction
	for _, acct := range accounts {
		l1Info.GenerateAccount(acct)
		faucetTxs = append(faucetTxs, l1Info.PrepareTx("faucet", acct, 30000, big.NewInt(1e14), nil))
	}
	SendWaitTestTransactions(t, ctx, l1Client, faucetTxs)

	for i := 1; i < seqInboxTestIters; i++ {
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
			for j := uint64(0); j < 70; j++ {
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
			Require(t, err)
			if currentHeader.Number.Int64()-int64(reorgTargetNumber) < 65 {
				Fail(t, "Less than 65 blocks of difference between current block", currentHeader.Number, "and target", reorgTargetNumber)
			}
			t.Logf("Reorganizing to L1 block %v", reorgTargetNumber)
			reorgTarget := l1BlockChain.GetBlockByNumber(reorgTargetNumber)
			err = l1BlockChain.ReorgToOldBlock(reorgTarget)
			Require(t, err)
			blockStates = blockStates[:(reorgTo + 1)]

			// Geth's miner's mempool might not immediately process the reorg.
			// Sometimes, this causes it to drop the next tx.
			// To work around this, we create a sacrificial tx, which may or may not succeed.
			// Whichever happens, by the end of this block, the miner will have processed the reorg.
			tx := l1Info.PrepareTx(fmt.Sprintf("ReorgSacrifice%v", i/10), "faucet", 30000, big.NewInt(0), nil)
			err = l1Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, _ = arbnode.WaitForTx(ctx, l1Client, tx.Hash(), time.Second)
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
					if !l2Info.HasAccount(name) {
						l2Info.GenerateAccount(name)
					}
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
				Require(t, err)
				var segment []byte
				segment = append(segment, arbstate.BatchSegmentKindL2Message)
				segment = append(segment, arbos.L2MessageKind_SignedTx)
				segment = append(segment, txData...)
				err = rlp.Encode(batchWriter, segment)
				Require(t, err)

				state.balances[source] -= amount
				state.balances[dest] += amount
			}

			Require(t, batchWriter.Close())
			batchData := batchBuffer.Bytes()

			seqNonce := len(blockStates) - 1
			for j := 0; ; j++ {
				haveNonce, err := l1Client.PendingNonceAt(ctx, seqOpts.From)
				Require(t, err)
				if haveNonce == uint64(seqNonce) {
					break
				}
				if j >= 10 {
					t.Fatal("timed out with sequencer nonce", haveNonce, "waiting for expected nonce", seqNonce)
				}
				time.Sleep(time.Millisecond * 100)
			}
			seqOpts.Nonce = big.NewInt(int64(seqNonce))
			var tx *types.Transaction
			if i%5 == 0 {
				tx, err = seqInbox.AddSequencerL2Batch(&seqOpts, big.NewInt(int64(len(blockStates)-1)), batchData, big.NewInt(0), common.Address{})
			} else {
				tx, err = seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(int64(len(blockStates)-1)), batchData, big.NewInt(0), common.Address{})
			}
			Require(t, err)
			txRes, err := arbnode.EnsureTxSucceeded(ctx, l1Client, tx)
			if err != nil {
				// Geth's clique miner is finicky.
				// Specifically, I suspect there's a race where it thinks there's no txs to put in the new block,
				// if a new tx arrives at the same time as it tries to create a block.
				// Resubmit the transaction in an attempt to get the miner going again.
				t.Error("had to resubmit tx")
				_ = l1Client.SendTransaction(ctx, tx)
				txRes, err = arbnode.EnsureTxSucceeded(ctx, l1Client, tx)
				Require(t, err)
			}

			state.l2BlockNumber += uint64(numMessages)
			state.l1BlockNumber = txRes.BlockNumber.Uint64()
			blockStates = append(blockStates, state)
		}

		t.Logf("Iteration %v: state %v block %v", i, len(blockStates)-1, blockStates[len(blockStates)-1].l2BlockNumber)

		for i := 0; ; i++ {
			batchCount, err := seqInbox.BatchCount(&bind.CallOpts{})
			if err != nil {
				Fail(t, err)
			}
			if batchCount.Cmp(big.NewInt(int64(len(blockStates)-1))) == 0 {
				break
			} else if i >= 100 {
				Fail(t, "timed out waiting for l1 batch count update; have", batchCount, "want", len(blockStates)-1)
			}
			time.Sleep(10 * time.Millisecond)
		}

		expectedBlockNumber := blockStates[len(blockStates)-1].l2BlockNumber
		for i := 0; ; i++ {
			blockNumber := l2Backend.APIBackend().CurrentHeader().Number.Uint64()
			if blockNumber == expectedBlockNumber {
				break
			} else if i >= 1000 {
				Fail(t, "timed out waiting for l2 block update; have", blockNumber, "want", expectedBlockNumber)
			}
			time.Sleep(10 * time.Millisecond)
		}

		for _, state := range blockStates {
			block, err := l2Backend.APIBackend().BlockByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			if block == nil {
				Fail(t, "missing state block", state.l2BlockNumber)
			}
			stateDb, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			for acct, balance := range state.balances {
				haveBalance := stateDb.GetBalance(acct)
				if new(big.Int).SetUint64(balance).Cmp(haveBalance) < 0 {
					Fail(t, "unexpected balance for account", acct, "; expected", balance, "got", haveBalance)
				}
			}
		}
	}
}
