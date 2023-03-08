// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util"
)

type blockTestState struct {
	balances      map[common.Address]*big.Int
	nonces        map[common.Address]uint64
	accounts      []common.Address
	l2BlockNumber uint64
	l1BlockNumber uint64
}

const seqInboxTestIters = 40

func testSequencerInboxReaderImpl(t *testing.T, validator bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conf := arbnode.ConfigDefaultL1Test()
	conf.InboxReader.HardReorg = true
	if validator {
		conf.BlockValidator.Enable = true
	}
	l2Info, arbNode, _, l1Info, l1backend, l1Client, l1stack := createTestNodeOnL1WithConfig(t, ctx, false, conf, nil, nil)
	l2Backend := arbNode.Backend
	defer requireClose(t, l1stack)
	defer arbNode.StopAndWait()

	l1BlockChain := l1backend.BlockChain()

	seqInbox, err := bridgegen.NewSequencerInbox(l1Info.GetAddress("SequencerInbox"), l1Client)
	Require(t, err)
	seqOpts := l1Info.GetDefaultTransactOpts("Sequencer", ctx)

	ownerAddress := l2Info.GetAddress("Owner")
	var startL2BlockNumber uint64 = 0

	startState, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	Require(t, err)
	startOwnerBalance := startState.GetBalance(ownerAddress)
	startOwnerNonce := startState.GetNonce(ownerAddress)

	var blockStates []blockTestState
	blockStates = append(blockStates, blockTestState{
		balances: map[common.Address]*big.Int{
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
		faucetTxs = append(faucetTxs, l1Info.PrepareTx("Faucet", acct, 30000, big.NewInt(1e16), nil))
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
					GasFeeCap: big.NewInt(params.GWei * 100),
					Value:     new(big.Int),
					Nonce:     j,
				}
				tx := l1Info.SignTxAs("ReorgPadding", rawTx)
				Require(t, l1Client.SendTransaction(ctx, tx))
				_, _ = EnsureTxSucceeded(ctx, l1Client, tx)
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
			tx := l1Info.PrepareTx(fmt.Sprintf("ReorgSacrifice%v", i/10), "Faucet", 30000, big.NewInt(0), nil)
			err = l1Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, _ = WaitForTx(ctx, l1Client, tx.Hash(), time.Second)
		} else {
			state := blockStates[len(blockStates)-1]
			newBalances := make(map[common.Address]*big.Int)
			for k, v := range state.balances {
				newBalances[k] = new(big.Int).Set(v)
			}
			state.balances = newBalances
			newNonces := make(map[common.Address]uint64)
			for k, v := range state.nonces {
				newNonces[k] = v
			}
			state.nonces = newNonces

			batchBuffer := bytes.NewBuffer([]byte{})
			numMessages := 1 + rand.Int()%5
			for j := 0; j < numMessages; j++ {
				sourceNum := rand.Int() % len(state.accounts)
				source := state.accounts[sourceNum]
				amount := new(big.Int).SetUint64(uint64(rand.Int()) % state.balances[source].Uint64())
				reserveAmount := new(big.Int).SetUint64(l2pricing.InitialBaseFeeWei * 100000000)
				if state.balances[source].Cmp(new(big.Int).Add(amount, reserveAmount)) < 0 {
					// Leave enough funds for gas
					amount = big.NewInt(1)
				}
				var dest common.Address
				if j == 0 && amount.Cmp(reserveAmount) >= 0 {
					name := accountName(len(state.accounts))
					if !l2Info.HasAccount(name) {
						l2Info.GenerateAccount(name)
					}
					dest = l2Info.GetAddress(name)
					state.accounts = append(state.accounts, dest)
					state.balances[dest] = big.NewInt(0)
				} else {
					dest = state.accounts[rand.Int()%len(state.accounts)]
				}

				rawTx := &types.DynamicFeeTx{
					To:        &dest,
					Gas:       util.NormalizeL2GasForL1GasInitial(210000, params.GWei),
					GasFeeCap: big.NewInt(l2pricing.InitialBaseFeeWei * 2),
					Value:     amount,
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
				err = rlp.Encode(batchBuffer, segment)
				Require(t, err)

				state.balances[source].Sub(state.balances[source], amount)
				state.balances[dest].Add(state.balances[dest], amount)
			}

			compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
			Require(t, err)
			batchData := append([]byte{0}, compressed...)

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
				tx, err = seqInbox.AddSequencerL2Batch(&seqOpts, big.NewInt(int64(len(blockStates))), batchData, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
			} else {
				tx, err = seqInbox.AddSequencerL2BatchFromOrigin(&seqOpts, big.NewInt(int64(len(blockStates))), batchData, big.NewInt(1), common.Address{})
			}
			Require(t, err)
			txRes, err := EnsureTxSucceeded(ctx, l1Client, tx)
			if err != nil {
				// Geth's clique miner is finicky.
				// Unfortunately this is so rare that I haven't had an opportunity to test this workaround.
				// Specifically, I suspect there's a race where it thinks there's no txs to put in the new block,
				// if a new tx arrives at the same time as it tries to create a block.
				// Resubmit the transaction in an attempt to get the miner going again.
				_ = l1Client.SendTransaction(ctx, tx)
				txRes, err = EnsureTxSucceeded(ctx, l1Client, tx)
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
			if batchCount.Cmp(big.NewInt(int64(len(blockStates)))) == 0 {
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

		if validator && i%15 == 0 {
			for i := 0; ; i++ {
				lastValidated := arbNode.BlockValidator.LastBlockValidated()
				if lastValidated == expectedBlockNumber {
					break
				} else if i >= 1000 {
					Fail(t, "timed out waiting for block validator; have", lastValidated, "want", expectedBlockNumber)
				}
				time.Sleep(time.Second)
			}
		}

		for _, state := range blockStates {
			block, err := l2Backend.APIBackend().BlockByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			if block == nil {
				Fail(t, "missing state block", state.l2BlockNumber)
			}
			stateDb, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			for acct, expectedBalance := range state.balances {
				haveBalance := stateDb.GetBalance(acct)
				if expectedBalance.Cmp(haveBalance) < 0 {
					Fail(t, "unexpected balance for account", acct, "; expected", expectedBalance, "got", haveBalance)
				}
			}
		}
	}
}

func TestSequencerInboxReader(t *testing.T) {
	testSequencerInboxReaderImpl(t, false)
}
