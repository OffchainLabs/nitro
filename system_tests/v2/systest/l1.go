// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

// L1-facing scenario helpers: delayed inbox sends, L1 block advancement, and
// L1→L2 message lookup. Available whenever the scenario has a parent chain (env.L1 != nil).

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

// l1DelayBlocks is how many parent-chain blocks to mine so a delayed-inbox
// message clears its delay and gets sequenced.
const l1DelayBlocks = 30

// ParentChain returns the parent-chain handle. Fails the test if the scenario has no L1.
func (e *Env) ParentChain() *L1Handle {
	e.t.Helper()
	e.requireL1()
	return e.L1
}

// DelayedInbox binds the deployed L1 Inbox contract.
func (e *Env) DelayedInbox() *bridgegen.Inbox {
	e.t.Helper()
	e.requireL1()
	inbox, err := bridgegen.NewInbox(e.L1.Info.GetAddress("Inbox"), e.L1.Client)
	e.Require(err, "NewInbox")
	return inbox
}

// AdvanceL1 mines n parent-chain blocks via no-op Faucet self-transfers.
func (e *Env) AdvanceL1(n int) {
	e.t.Helper()
	e.requireL1()
	for range n {
		tx := e.L1.Info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil)
		e.L1.SendTx(tx)
		e.L1.EnsureTxSucceeded(tx)
	}
}

// WaitForL1DelayBlocks mines enough L1 blocks to release pending delayed-inbox
// messages.
func (e *Env) WaitForL1DelayBlocks() {
	e.t.Helper()
	e.AdvanceL1(l1DelayBlocks)
}

// SendSignedTxViaL1 posts a signed L2 tx through the L1 delayed inbox, advances
// L1 past the delay, and waits for it to land on L2. Returns the L2 receipt.
func (e *Env) SendSignedTxViaL1(delayedTx *types.Transaction) *types.Receipt {
	e.t.Helper()
	e.requireL1()
	opts := e.L1.Info.GetDefaultTransactOpts("User", e.Ctx)
	txbytes, err := delayedTx.MarshalBinary()
	e.Require(err, "MarshalBinary")
	wrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	l1tx, err := e.DelayedInbox().SendL2Message(&opts, wrapped)
	e.Require(err, "SendL2Message")
	e.L1.EnsureTxSucceeded(l1tx)
	e.WaitForL1DelayBlocks()
	return e.L2.EnsureTxSucceeded(delayedTx)
}

// SendSignedTxBatchViaL1 posts a batch of signed L2 txs through the delayed
// inbox in one L1 message, advances L1, and waits for each on L2.
func (e *Env) SendSignedTxBatchViaL1(txes types.Transactions) types.Receipts {
	e.t.Helper()
	e.requireL1()
	opts := e.L1.Info.GetDefaultTransactOpts("User", e.Ctx)
	l1tx, err := e.DelayedInbox().SendL2Message(&opts, e.l2MessageBatchData(txes))
	e.Require(err, "SendL2Message batch")
	e.L1.EnsureTxSucceeded(l1tx)
	e.WaitForL1DelayBlocks()
	receipts := make(types.Receipts, 0, len(txes))
	for _, tx := range txes {
		receipts = append(receipts, e.L2.EnsureTxSucceeded(tx))
	}
	return receipts
}

// LookupL2Tx finds the single L2 submission transaction produced by an L1
// receipt (deposit / retryable / contract tx). Fails if not exactly one.
func (e *Env) LookupL2Tx(l1Receipt *types.Receipt) *types.Transaction {
	e.t.Helper()
	e.requireL1()
	bridge, err := arbnode.NewDelayedBridge(e.L1.Client, e.L1.Info.GetAddress("Bridge"), 0)
	e.Require(err, "NewDelayedBridge")
	messages, err := bridge.LookupMessagesInRange(e.Ctx, l1Receipt.BlockNumber, l1Receipt.BlockNumber, nil)
	e.Require(err, "LookupMessagesInRange")
	e.NotEmpty(messages, "LookupL2Tx: no message for submission")
	msgTypes := map[uint8]bool{
		arbostypes.L1MessageType_SubmitRetryable: true,
		arbostypes.L1MessageType_EthDeposit:      true,
		arbostypes.L1MessageType_L2Message:       true,
	}
	txTypes := map[uint8]bool{
		types.ArbitrumSubmitRetryableTxType: true,
		types.ArbitrumDepositTxType:         true,
		types.ArbitrumContractTxType:        true,
	}
	var submissionTxs []*types.Transaction
	chainID := e.L2.ChainID()
	for _, message := range messages {
		if !msgTypes[message.Message.Header.Kind] {
			continue
		}
		txs, err := arbos.ParseL2Transactions(message.Message, chainID, params.MaxDebugArbosVersionSupported)
		e.Require(err, "ParseL2Transactions")
		for _, tx := range txs {
			if txTypes[tx.Type()] {
				submissionTxs = append(submissionTxs, tx)
			}
		}
	}
	e.Len(submissionTxs, 1, "LookupL2Tx: expected exactly 1 submission tx")
	return submissionTxs[0]
}

// requireL1 fails the scenario with a clear message if it lacks a parent chain.
func (e *Env) requireL1() {
	e.t.Helper()
	e.NotNil(e.L1, "L1 helper called on a non-L1 scenario; register it with systest.WithL1()")
}

func (e *Env) l2MessageBatchData(txes types.Transactions) []byte {
	e.t.Helper()
	l2Message := []byte{arbos.L2MessageKind_Batch}
	sizeBuf := make([]byte, 8)
	for _, tx := range txes {
		txBytes, err := tx.MarshalBinary()
		e.Require(err, "MarshalBinary")
		binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes))+1)
		l2Message = append(l2Message, sizeBuf...)
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	}
	return l2Message
}
