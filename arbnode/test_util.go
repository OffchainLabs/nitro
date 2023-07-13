// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"errors"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

type execClientWrapper struct {
	*gethexec.ExecutionEngine
	t *testing.T
}

func (w *execClientWrapper) Pause() containers.PromiseInterface[struct{}] {
	w.t.Error("not supported")
	return containers.NewReadyPromise[struct{}](struct{}{}, errors.New("Pause not supported"))
}

func (w *execClientWrapper) Activate() containers.PromiseInterface[struct{}] {
	w.t.Error("not supported")
	return containers.NewReadyPromise[struct{}](struct{}{}, errors.New("Activate not supported"))
}

func (w *execClientWrapper) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	w.t.Error("not supported")
	return containers.NewReadyPromise[struct{}](struct{}{}, errors.New("ForwardTo not supported"))
}

func NewTransactionStreamerForTest(t *testing.T, ownerAddress common.Address) (*gethexec.ExecutionEngine, *TransactionStreamer, ethdb.Database, *core.BlockChain) {
	chainConfig := params.ArbitrumDevTestChainConfig()

	initData := statetransfer.ArbosInitializationInfo{
		Accounts: []statetransfer.AccountInitializationInfo{
			{
				Addr:       ownerAddress,
				EthBalance: big.NewInt(params.Ether),
			},
		},
	}

	chainDb := rawdb.NewMemoryDatabase()
	arbDb := rawdb.NewMemoryDatabase()
	execDb := rawdb.NewMemoryDatabase()
	initReader := statetransfer.NewMemoryInitDataReader(&initData)

	bc, err := gethexec.WriteOrTestBlockChain(chainDb, nil, initReader, chainConfig, arbostypes.TestInitMessage, gethexec.ConfigDefaultTest().TxLookupLimit, 0)

	if err != nil {
		Fail(t, err)
	}

	transactionStreamerConfigFetcher := func() *TransactionStreamerConfig { return &DefaultTransactionStreamerConfig }
	execEngine, err := gethexec.NewExecutionEngine(bc, execDb, nil)
	if err != nil {
		Fail(t, err)
	}
	execSeq := &execClientWrapper{execEngine, t}
	inbox, err := NewTransactionStreamer(arbDb, bc.Config(), execSeq, nil, make(chan error, 1), transactionStreamerConfigFetcher)
	if err != nil {
		Fail(t, err)
	}

	// Add the init message
	err = inbox.AddFakeInitMessage()
	if err != nil {
		Fail(t, err)
	}

	return execEngine, inbox, arbDb, bc
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
