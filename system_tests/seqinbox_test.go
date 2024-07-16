// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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

func encodeAddBatch(seqABI *abi.ABI, seqNum *big.Int, message []byte, afterDelayedMsgRead *big.Int, gasRefunder common.Address) ([]byte, error) {
	method, ok := seqABI.Methods["addSequencerL2BatchFromOrigin0"]
	if !ok {
		return nil, errors.New("failed to find add addSequencerL2BatchFromOrigin0 method")
	}
	inputData, err := method.Inputs.Pack(
		seqNum,
		message,
		afterDelayedMsgRead,
		gasRefunder,
		new(big.Int).SetUint64(uint64(1)),
		new(big.Int).SetUint64(uint64(1)),
	)
	if err != nil {
		return nil, err
	}
	fullData := append([]byte{}, method.ID...)
	fullData = append(fullData, inputData...)
	return fullData, nil
}
func diffAccessList(accessed, al types.AccessList) string {
	m := make(map[common.Address]map[common.Hash]bool)
	for i := 0; i < len(al); i++ {
		if _, ok := m[al[i].Address]; !ok {
			m[al[i].Address] = make(map[common.Hash]bool)
		}
		for _, slot := range al[i].StorageKeys {
			m[al[i].Address][slot] = true
		}
	}

	diff := ""
	for i := 0; i < len(accessed); i++ {
		addr := accessed[i].Address
		if _, ok := m[addr]; !ok {
			diff += fmt.Sprintf("contract address: %q wasn't accessed\n", addr)
			continue
		}
		for j := 0; j < len(accessed[i].StorageKeys); j++ {
			slot := accessed[i].StorageKeys[j]
			if _, ok := m[addr][slot]; !ok {
				diff += fmt.Sprintf("storage slot: %v for contract: %v wasn't accessed\n", slot, addr)
			}
		}
	}
	return diff
}

func deployGasRefunder(ctx context.Context, t *testing.T, builder *NodeBuilder) common.Address {
	t.Helper()
	abi, err := bridgegen.GasRefunderMetaData.GetAbi()
	if err != nil {
		t.Fatalf("Error getting gas refunder abi: %v", err)
	}
	fauOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)
	addr, tx, _, err := bind.DeployContract(&fauOpts, *abi, common.FromHex(bridgegen.GasRefunderBin), builder.L1.Client)
	if err != nil {
		t.Fatalf("Error getting gas refunder contract deployment transaction: %v", err)
	}
	if _, err := builder.L1.EnsureTxSucceeded(tx); err != nil {
		t.Fatalf("Error deploying gas refunder contract: %v", err)
	}
	tx = builder.L1Info.PrepareTxTo("Faucet", &addr, 30000, big.NewInt(9223372036854775807), nil)
	if err := builder.L1.Client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("Error sending gas refunder funding transaction")
	}
	if _, err := builder.L1.EnsureTxSucceeded(tx); err != nil {
		t.Fatalf("Error funding gas refunder")
	}
	contract, err := bridgegen.NewGasRefunder(addr, builder.L1.Client)
	if err != nil {
		t.Fatalf("Error getting gas refunder contract binding: %v", err)
	}
	tx, err = contract.AllowContracts(&fauOpts, []common.Address{builder.L1Info.GetAddress("SequencerInbox")})
	if err != nil {
		t.Fatalf("Error creating transaction for altering allowlist in refunder: %v", err)
	}
	if _, err := builder.L1.EnsureTxSucceeded(tx); err != nil {
		t.Fatalf("Error addting sequencer inbox in gas refunder allowlist: %v", err)
	}

	tx, err = contract.AllowRefundees(&fauOpts, []common.Address{builder.L1Info.GetAddress("Sequencer")})
	if err != nil {
		t.Fatalf("Error creating transaction for altering allowlist in refunder: %v", err)
	}
	if _, err := builder.L1.EnsureTxSucceeded(tx); err != nil {
		t.Fatalf("Error addting sequencer in gas refunder allowlist: %v", err)
	}
	return addr
}

func testSequencerInboxReaderImpl(t *testing.T, validator bool) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.InboxReader.HardReorg = true
	if validator {
		builder.nodeConfig.BlockValidator.Enable = true
	}
	builder.isSequencer = false
	cleanup := builder.Build(t)
	defer cleanup()

	l2Backend := builder.L2.ExecNode.Backend

	l1BlockChain := builder.L1.L1Backend.BlockChain()

	rpcC := builder.L1.Stack.Attach()
	gethClient := gethclient.New(rpcC)

	seqInbox, err := bridgegen.NewSequencerInbox(builder.L1Info.GetAddress("SequencerInbox"), builder.L1.Client)
	Require(t, err)
	seqOpts := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)

	gasRefunderAddr := deployGasRefunder(ctx, t, builder)

	ownerAddress := builder.L2Info.GetAddress("Owner")
	var startL2BlockNumber uint64 = 0

	startState, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	Require(t, err)
	startOwnerBalance := startState.GetBalance(ownerAddress)
	startOwnerNonce := startState.GetNonce(ownerAddress)

	var blockStates []blockTestState
	blockStates = append(blockStates, blockTestState{
		balances: map[common.Address]*big.Int{
			ownerAddress: startOwnerBalance.ToBig(),
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
		}
		return fmt.Sprintf("Account%v", x)
	}

	accounts := []string{"ReorgPadding"}
	for i := 1; i <= (seqInboxTestIters-1)/10; i++ {
		accounts = append(accounts, fmt.Sprintf("ReorgSacrifice%v", i))
	}
	var faucetTxs []*types.Transaction
	for _, acct := range accounts {
		builder.L1Info.GenerateAccount(acct)
		faucetTxs = append(faucetTxs, builder.L1Info.PrepareTx("Faucet", acct, 30000, big.NewInt(1e16), nil))
	}
	builder.L1.SendWaitTestTransactions(t, faucetTxs)

	seqABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		t.Fatalf("Error getting sequencer inbox abi: %v", err)
	}

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
			padAddr := builder.L1Info.GetAddress("ReorgPadding")
			for j := uint64(0); j < 70; j++ {
				rawTx := &types.DynamicFeeTx{
					To:        &padAddr,
					Gas:       21000,
					GasFeeCap: big.NewInt(params.GWei * 100),
					Value:     new(big.Int),
					Nonce:     j,
				}
				tx := builder.L1Info.SignTxAs("ReorgPadding", rawTx)
				Require(t, builder.L1.Client.SendTransaction(ctx, tx))
				_, _ = builder.L1.EnsureTxSucceeded(tx)
			}
			reorgTargetNumber := blockStates[reorgTo].l1BlockNumber
			currentHeader, err := builder.L1.Client.HeaderByNumber(ctx, nil)
			Require(t, err)
			if currentHeader.Number.Int64()-int64(reorgTargetNumber) < 65 {
				Fatal(t, "Less than 65 blocks of difference between current block", currentHeader.Number, "and target", reorgTargetNumber)
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
			tx := builder.L1Info.PrepareTx(fmt.Sprintf("ReorgSacrifice%v", i/10), "Faucet", 30000, big.NewInt(0), nil)
			err = builder.L1.Client.SendTransaction(ctx, tx)
			Require(t, err)
			_, _ = WaitForTx(ctx, builder.L1.Client, tx.Hash(), time.Second)
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
					if !builder.L2Info.HasAccount(name) {
						builder.L2Info.GenerateAccount(name)
					}
					dest = builder.L2Info.GetAddress(name)
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
				tx := builder.L2Info.SignTxAs(accountName(sourceNum), rawTx)
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
				haveNonce, err := builder.L1.Client.PendingNonceAt(ctx, seqOpts.From)
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
			before, err := builder.L1.Client.BalanceAt(ctx, seqOpts.From, nil)
			if err != nil {
				t.Fatalf("BalanceAt(%v) unexpected error: %v", seqOpts.From, err)
			}

			data, err := encodeAddBatch(seqABI, big.NewInt(int64(len(blockStates))), batchData, big.NewInt(1), gasRefunderAddr)
			if err != nil {
				t.Fatalf("Error encoding batch data: %v", err)
			}
			si := builder.L1Info.GetAddress("SequencerInbox")
			wantAL, _, _, err := gethClient.CreateAccessList(ctx, ethereum.CallMsg{
				From: seqOpts.From,
				To:   &si,
				Data: data,
			})
			if err != nil {
				t.Fatalf("Error creating access list: %v", err)
			}
			accessed := arbnode.AccessList(&arbnode.AccessListOpts{
				SequencerInboxAddr:       builder.L1Info.GetAddress("SequencerInbox"),
				BridgeAddr:               builder.L1Info.GetAddress("Bridge"),
				DataPosterAddr:           seqOpts.From,
				GasRefunderAddr:          gasRefunderAddr,
				SequencerInboxAccs:       len(blockStates),
				AfterDelayedMessagesRead: 1,
			})
			if diff := diffAccessList(accessed, *wantAL); diff != "" {
				t.Errorf("Access list mismatch:\n%s\n", diff)
			}
			if i%5 == 0 {
				tx, err = seqInbox.AddSequencerL2Batch(&seqOpts, big.NewInt(int64(len(blockStates))), batchData, big.NewInt(1), gasRefunderAddr, big.NewInt(0), big.NewInt(0))
			} else {
				tx, err = seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(&seqOpts, big.NewInt(int64(len(blockStates))), batchData, big.NewInt(1), gasRefunderAddr, common.Big0, common.Big0)
			}
			Require(t, err)
			txRes, err := builder.L1.EnsureTxSucceeded(tx)
			if err != nil {
				// Geth's clique miner is finicky.
				// Unfortunately this is so rare that I haven't had an opportunity to test this workaround.
				// Specifically, I suspect there's a race where it thinks there's no txs to put in the new block,
				// if a new tx arrives at the same time as it tries to create a block.
				// Resubmit the transaction in an attempt to get the miner going again.
				_ = builder.L1.Client.SendTransaction(ctx, tx)
				txRes, err = builder.L1.EnsureTxSucceeded(tx)
				Require(t, err)
			}
			after, err := builder.L1.Client.BalanceAt(ctx, seqOpts.From, nil)
			if err != nil {
				t.Fatalf("BalanceAt(%v) unexpected error: %v", seqOpts.From, err)
			}
			txCost := txRes.EffectiveGasPrice.Uint64() * txRes.GasUsed
			if diff := before.Int64() - after.Int64(); diff >= int64(txCost) {
				t.Errorf("Transaction: %v was not refunded, balance diff: %v, cost: %v", tx.Hash(), diff, txCost)
			}

			state.l2BlockNumber += uint64(numMessages)
			state.l1BlockNumber = txRes.BlockNumber.Uint64()
			blockStates = append(blockStates, state)
		}

		t.Logf("Iteration %v: state %v block %v", i, len(blockStates)-1, blockStates[len(blockStates)-1].l2BlockNumber)

		for i := 0; ; i++ {
			batchCount, err := seqInbox.BatchCount(&bind.CallOpts{})
			if err != nil {
				Fatal(t, err)
			}
			if batchCount.Cmp(big.NewInt(int64(len(blockStates)))) == 0 {
				break
			} else if i >= 140 {
				Fatal(t, "timed out waiting for l1 batch count update; have", batchCount, "want", len(blockStates)-1)
			}
			time.Sleep(10 * time.Millisecond)
		}

		expectedBlockNumber := blockStates[len(blockStates)-1].l2BlockNumber
		for i := 0; ; i++ {
			blockNumber := l2Backend.APIBackend().CurrentHeader().Number.Uint64()
			if blockNumber == expectedBlockNumber {
				break
			} else if i >= 1000 {
				Fatal(t, "timed out waiting for l2 block update; have", blockNumber, "want", expectedBlockNumber)
			}
			time.Sleep(10 * time.Millisecond)
		}

		if validator && i%15 == 0 {
			for i := 0; ; i++ {
				expectedPos, err := builder.L2.ExecNode.ExecEngine.BlockNumberToMessageIndex(expectedBlockNumber)
				Require(t, err)
				lastValidated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
				if lastValidated == expectedPos+1 {
					break
				} else if i >= 1000 {
					Fatal(t, "timed out waiting for block validator; have", lastValidated, "want", expectedPos+1)
				}
				time.Sleep(time.Second)
			}
		}

		for _, state := range blockStates {
			block, err := l2Backend.APIBackend().BlockByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			if block == nil {
				Fatal(t, "missing state block", state.l2BlockNumber)
			}
			stateDb, _, err := l2Backend.APIBackend().StateAndHeaderByNumber(ctx, rpc.BlockNumber(state.l2BlockNumber))
			Require(t, err)
			for acct, expectedBalance := range state.balances {
				haveBalance := stateDb.GetBalance(acct)
				if expectedBalance.Cmp(haveBalance.ToBig()) < 0 {
					Fatal(t, "unexpected balance for account", acct, "; expected", expectedBalance, "got", haveBalance)
				}
			}
		}
	}
}

func TestSequencerInboxReader(t *testing.T) {
	t.Skip("diagnose after Stylus merge")
	testSequencerInboxReaderImpl(t, false)
}
