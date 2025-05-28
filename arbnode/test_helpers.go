package arbnode

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

type execClientWrapper struct {
	ExecutionEngine *gethexec.ExecutionEngine
	t               *testing.T
}

func (w *execClientWrapper) Pause() { w.t.Error("not supported") }

func (w *execClientWrapper) Activate() { w.t.Error("not supported") }

func (w *execClientWrapper) ForwardTo(url string) error { w.t.Error("not supported"); return nil }

func (w *execClientWrapper) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	return w.ExecutionEngine.SequenceDelayedMessage(message, delayedSeqNum)
}

func (w *execClientWrapper) NextDelayedMessageNumber() (uint64, error) {
	return w.ExecutionEngine.NextDelayedMessageNumber()
}

func (w *execClientWrapper) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	markFeedStartWithReturn := func(to arbutil.MessageIndex) (struct{}, error) {
		w.ExecutionEngine.MarkFeedStart(to)
		return struct{}{}, nil
	}
	return containers.NewReadyPromise(markFeedStartWithReturn(to))
}

func (w *execClientWrapper) Maintenance() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (w *execClientWrapper) Synced(ctx context.Context) bool {
	w.t.Error("not supported")
	return false
}
func (w *execClientWrapper) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	w.t.Error("not supported")
	return nil
}
func (w *execClientWrapper) SetFinalityData(
	ctx context.Context,
	safeFinalityData *arbutil.FinalityData,
	finalizedFinalityData *arbutil.FinalityData,
	validatedFinalityData *arbutil.FinalityData,
) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (w *execClientWrapper) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return containers.NewReadyPromise(w.ExecutionEngine.DigestMessage(num, msg, msgForPrefetch))
}

func (w *execClientWrapper) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	return containers.NewReadyPromise(w.ExecutionEngine.Reorg(count, newMessages, oldMessages))
}

func (w *execClientWrapper) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise(w.ExecutionEngine.HeadMessageIndex())
}

func (w *execClientWrapper) ResultAtMessageIndex(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return containers.NewReadyPromise(w.ExecutionEngine.ResultAtMessageIndex(pos))
}

func (w *execClientWrapper) Start(ctx context.Context) error {
	return nil
}

func (w *execClientWrapper) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise(w.ExecutionEngine.MessageIndexToBlockNumber(messageNum), nil)
}

func (w *execClientWrapper) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise(w.ExecutionEngine.BlockNumberToMessageIndex(blockNum))
}

func (w *execClientWrapper) StopAndWait() {
}

func NewTransactionStreamerForTest(t *testing.T, ctx context.Context, ownerAddress common.Address) (*gethexec.ExecutionEngine, *TransactionStreamer, ethdb.Database, *core.BlockChain) {
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()

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
	initReader := statetransfer.NewMemoryInitDataReader(&initData)

	cacheConfig := core.DefaultCacheConfigWithScheme(env.GetTestStateScheme())
	bc, err := gethexec.WriteOrTestBlockChain(chainDb, cacheConfig, initReader, chainConfig, nil, nil, arbostypes.TestInitMessage, gethexec.ConfigDefault.TxLookupLimit, 0)

	if err != nil {
		Fail(t, err)
	}

	transactionStreamerConfigFetcher := func() *TransactionStreamerConfig { return &DefaultTransactionStreamerConfig }
	execEngine, err := gethexec.NewExecutionEngine(bc, 0)
	if err != nil {
		Fail(t, err)
	}
	stylusTargetConfig := &gethexec.DefaultStylusTargetConfig
	Require(t, stylusTargetConfig.Validate()) // pre-processes config (i.a. parses wasmTargets)
	if err := execEngine.Initialize(gethexec.DefaultCachingConfig.StylusLRUCacheCapacity, &gethexec.DefaultStylusTargetConfig); err != nil {
		Fail(t, err)
	}
	execSeq := &execClientWrapper{execEngine, t}
	inbox, err := NewTransactionStreamer(ctx, arbDb, bc.Config(), execSeq, nil, make(chan error, 1), transactionStreamerConfigFetcher, &DefaultSnapSyncConfig)
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
