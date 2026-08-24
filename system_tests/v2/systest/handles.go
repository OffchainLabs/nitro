// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/containers"
)

// ChainHandle is the client+info surface shared by every layer (L1, L2).
type ChainHandle struct {
	// Default in-process client (via stack.Attach). Always populated.
	Client *ethclient.Client
	// Info carries the chain's test accounts and deployed contract addresses.
	Info *arbtest.BlockchainTestInfo

	e    *Env
	name string
}

func (c *ChainHandle) TransactOpts(name string) bind.TransactOpts {
	return c.Info.GetDefaultTransactOpts(name, c.e.Ctx)
}

func (c *ChainHandle) BalanceAt(addr common.Address) *big.Int {
	c.e.t.Helper()
	bal, err := c.Client.BalanceAt(c.e.Ctx, addr, nil)
	c.e.Require(err, "%s BalanceAt %v", c.name, addr)
	return bal
}

func (c *ChainHandle) ChainID() *big.Int {
	c.e.t.Helper()
	id, err := c.Client.ChainID(c.e.Ctx)
	c.e.Require(err, "%s ChainID", c.name)
	return id
}

func (c *ChainHandle) HeaderByNumber(number *big.Int) *types.Header {
	c.e.t.Helper()
	header, err := c.Client.HeaderByNumber(c.e.Ctx, number)
	c.e.Require(err, "%s HeaderByNumber %v", c.name, number)
	return header
}

func (c *ChainHandle) PendingNonceAt(addr common.Address) uint64 {
	c.e.t.Helper()
	nonce, err := c.Client.PendingNonceAt(c.e.Ctx, addr)
	c.e.Require(err, "%s PendingNonceAt %v", c.name, addr)
	return nonce
}

func (c *ChainHandle) CodeAt(addr common.Address, blockNumber *big.Int) []byte {
	c.e.t.Helper()
	code, err := c.Client.CodeAt(c.e.Ctx, addr, blockNumber)
	c.e.Require(err, "%s CodeAt %v", c.name, addr)
	return code
}

func (c *ChainHandle) CallContract(msg ethereum.CallMsg, blockNumber *big.Int) []byte {
	c.e.t.Helper()
	ret, err := c.Client.CallContract(c.e.Ctx, msg, blockNumber)
	c.e.Require(err, "%s CallContract", c.name)
	return ret
}

func (c *ChainHandle) EstimateGas(msg ethereum.CallMsg) uint64 {
	c.e.t.Helper()
	gas, err := c.Client.EstimateGas(c.e.Ctx, msg)
	c.e.Require(err, "%s EstimateGas", c.name)
	return gas
}

// EnsureTxSucceeded polls until tx is included on this chain, up to
// DefaultTxWaitTimeout. Fails the test on timeout or revert.
func (c *ChainHandle) EnsureTxSucceeded(tx *types.Transaction) *types.Receipt {
	c.e.t.Helper()
	receipt, err := ensureTxSucceededWithin(c.e.Ctx, c.Client, tx, DefaultTxWaitTimeout)
	c.e.Require(err, "%s EnsureTxSucceeded", c.name)
	return receipt
}

// SendTx submits tx to this chain, failing the test if the send is rejected.
func (c *ChainHandle) SendTx(tx *types.Transaction) {
	c.e.t.Helper()
	c.e.Require(c.Client.SendTransaction(c.e.Ctx, tx), "%s send tx", c.name)
}

// EnsureTxFailed waits for tx to be mined and fails the test unless it reverted.
func (c *ChainHandle) EnsureTxFailed(tx *types.Transaction) *types.Receipt {
	c.e.t.Helper()
	receipt, err := waitForTxWithTimeout(c.e.Ctx, c.Client, tx.Hash(), DefaultTxWaitTimeout)
	c.e.Require(err, "%s EnsureTxFailed wait tx %s", c.name, tx.Hash())
	if receipt.Status != types.ReceiptStatusFailed {
		c.e.Require(fmt.Errorf("%s transaction %s unexpectedly succeeded", c.name, tx.Hash()))
	}
	return receipt
}

// WaitForTx waits for tx's receipt with no success checks.
func (c *ChainHandle) WaitForTx(tx *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	return waitForTxWithTimeout(c.e.Ctx, c.Client, tx.Hash(), timeout)
}

// AdvanceBlocks emits n no-op Faucet self-transfers, waiting each to mine.
func (c *ChainHandle) AdvanceBlocks(n int) {
	c.e.t.Helper()
	for range n {
		tx := c.Info.PrepareTx("Faucet", "Faucet", c.Info.TransferGas, common.Big1, nil)
		c.SendTx(tx)
		c.EnsureTxSucceeded(tx)
	}
}

// advanceBlock mines one block via a no-op Faucet self-transfer without
// failing the test, so polling loops keep their own error reporting.
func (c *ChainHandle) advanceBlock(ctx context.Context) error {
	tx := c.Info.PrepareTx("Faucet", "Faucet", c.Info.TransferGas, common.Big1, nil)
	if err := c.Client.SendTransaction(ctx, tx); err != nil {
		return err
	}
	_, err := ensureTxSucceededWithin(ctx, c.Client, tx, DefaultTxWaitTimeout)
	return err
}

// TransferBalance sends amount from->to and waits for success.
func (c *ChainHandle) TransferBalance(from, to string, amount *big.Int) (*types.Transaction, *types.Receipt) {
	c.e.t.Helper()
	tx := c.Info.PrepareTx(from, to, c.Info.TransferGas, amount, nil)
	c.SendTx(tx)
	return tx, c.EnsureTxSucceeded(tx)
}

// SendWaitTestTransactions submits all txs, then waits for each.
func (c *ChainHandle) SendWaitTestTransactions(txs []*types.Transaction) []*types.Receipt {
	c.e.t.Helper()
	receipts := make([]*types.Receipt, 0, len(txs))
	for _, tx := range txs {
		c.SendTx(tx)
	}
	for _, tx := range txs {
		receipts = append(receipts, c.EnsureTxSucceeded(tx))
	}
	return receipts
}

// L2Handle is the live L2 client plus low-level escape hatches for tests
// needing raw stack or exec-node access.
type L2Handle struct {
	ChainHandle

	// HTTP / WS clients populated when the stack exposes those endpoints.
	// Nil if endpoints aren't configured.
	HTTPClient *ethclient.Client
	WSClient   *ethclient.Client

	// Read-only escape hatches; geth-only. Prefer Client (eth JSON-RPC) for
	// client-agnostic access.
	Stack    *node.Node
	ExecNode *gethexec.ExecutionNode

	// Consensus is the L2 consensus node. Exposed so scenarios can reach the
	// parent-chain data source (batch counts/metadata) on L1 topologies.
	Consensus *arbnode.Node
}

// L1Handle is the live parent-chain client plus low-level escape hatches for
// tests that drive L1 directly (delayed inbox, deposits, reorgs).
type L1Handle struct {
	ChainHandle

	// Low-level escape hatches: Backend for forced reorgs, Stack for endpoints.
	Backend *eth.Ethereum
	Stack   *node.Node

	// Shared artifacts a second L2 node needs to follow this chain.
	blobReader containers.Option[daprovider.BlobReader]
	initMsg    *arbostypes.ParsedInitMessage
	wasmRoot   common.Hash
}

// EnsureTxSucceeded waits for tx to succeed, then until its block is tagged safe:
// the simulated beacon tags blocks safe asynchronously, and components that poll
// the safe tag would race a return at latest (v1 flake fix from 2023, kept for parity).
func (h *L1Handle) EnsureTxSucceeded(tx *types.Transaction) *types.Receipt {
	h.e.t.Helper()
	receipt := h.ChainHandle.EnsureTxSucceeded(tx)
	h.e.Require(waitForSafeBlock(h.e.Ctx, h.Client, receipt.BlockNumber, DefaultTxWaitTimeout), "%s wait safe block for tx %s", h.name, tx.Hash())
	return receipt
}

// l1DelayBlocks is how many parent-chain blocks to mine so a delayed-inbox
// message clears its delay and gets sequenced.
const l1DelayBlocks = 30

// DelayedInbox binds the deployed Inbox contract on this chain.
func (h *L1Handle) DelayedInbox() *bridgegen.Inbox {
	h.e.t.Helper()
	inbox, err := bridgegen.NewInbox(h.Info.GetAddress("Inbox"), h.Client)
	h.e.Require(err, "NewInbox")
	return inbox
}

// WaitForDelayBlocks mines enough blocks to release pending delayed-inbox
// messages.
func (h *L1Handle) WaitForDelayBlocks() {
	h.e.t.Helper()
	h.AdvanceBlocks(l1DelayBlocks)
}
