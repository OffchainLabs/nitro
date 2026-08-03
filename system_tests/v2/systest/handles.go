// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	arbtest "github.com/offchainlabs/nitro/system_tests"
)

// ChainHandle is the client+info surface shared by every layer (L1, L2, followers).
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

// SendWaitTxs sends count value-transfer txs from->to and waits for each.
func (c *ChainHandle) SendWaitTxs(from, to string, count int, value *big.Int) []*types.Receipt {
	c.e.t.Helper()
	receipts := make([]*types.Receipt, 0, count)
	for range count {
		tx := c.Info.PrepareTx(from, to, c.Info.TransferGas, value, nil)
		c.e.Require(c.Client.SendTransaction(c.e.Ctx, tx), "%s SendWaitTxs send", c.name)
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

	cleanup func()
}
