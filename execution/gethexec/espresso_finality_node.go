package gethexec

import (
	"context"
	"fmt"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-sequencer-go/client"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	retryTime = time.Second * 1
)

/*
Espresso Finality Node creates blocks with finalized hotshot transactions
*/
type EspressoFinalityNode struct {
	stopwaiter.StopWaiter

	config     SequencerConfigFetcher
	execEngine *ExecutionEngine
	namespace  uint64

	espressoClient  *espressoClient.Client
	nextSeqBlockNum uint64
}

func NewEspressoFinalityNode(execEngine *ExecutionEngine, configFetcher SequencerConfigFetcher) *EspressoFinalityNode {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		panic(err)
	}
	return &EspressoFinalityNode{
		execEngine:      execEngine,
		config:          configFetcher,
		namespace:       config.EspressoFinalityNodeConfig.Namespace,
		espressoClient:  espressoClient.NewClient(config.EspressoFinalityNodeConfig.HotShotUrl),
		nextSeqBlockNum: config.EspressoFinalityNodeConfig.StartBlock,
	}
}

func (n *EspressoFinalityNode) createBlock(ctx context.Context) (returnValue bool) {
	if n.nextSeqBlockNum == 0 {
		latestBlock, err := n.espressoClient.FetchLatestBlockHeight(ctx)
		if err != nil && latestBlock == 0 {
			log.Warn("unable to fetch latest hotshot block", "err", err)
			return false
		}
		log.Info("Started espresso finality node at the latest hotshot block", "block number", latestBlock)
		n.nextSeqBlockNum = latestBlock
	}

	nextSeqBlockNum := n.nextSeqBlockNum
	header, err := n.espressoClient.FetchHeaderByHeight(ctx, nextSeqBlockNum)
	if err != nil {
		arbos.LogFailedToFetchHeader(nextSeqBlockNum)
		return false
	}

	height := header.Header.GetBlockHeight()
	arbTxns, err := n.espressoClient.FetchTransactionsInBlock(ctx, height, n.namespace)
	if err != nil {
		arbos.LogFailedToFetchTransactions(height, err)
		return false
	}
	arbHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		Poster:      l1pricing.BatchPosterAddress,
		BlockNumber: header.Header.GetL1Head(),
		Timestamp:   header.Header.GetTimestamp(),
		RequestId:   nil,
		L1BaseFee:   nil,
	}

	// Deserialize the transactions and remove the signature from the transactions.
	// Ignore the malformed transactions
	txes := types.Transactions{}
	for _, tx := range arbTxns.Transactions {
		var out types.Transaction
		// signature from the data poster is the first 65 bytes of a transaction
		tx = tx[65:]
		if err := out.UnmarshalBinary(tx); err != nil {
			log.Warn("malformed tx found")
			continue
		}
		txes = append(txes, &out)
	}

	hooks := arbos.NoopSequencingHooks()
	_, err = n.execEngine.SequenceTransactions(arbHeader, txes, hooks)
	if err != nil {
		log.Error("espresso finality node: failed to sequence transactions", "err", err)
		return false
	}

	return true
}

func (n *EspressoFinalityNode) Start(ctx context.Context) error {
	n.StopWaiter.Start(ctx, n)
	err := n.CallIterativelySafe(func(ctx context.Context) time.Duration {
		madeBlock := n.createBlock(ctx)
		if madeBlock {
			n.nextSeqBlockNum += 1
			return 0
		}
		return retryTime
	})
	if err != nil {
		return fmt.Errorf("failed to start espresso finality node: %w", err)
	}
	return nil
}

func (n *EspressoFinalityNode) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return nil
}

func (n *EspressoFinalityNode) CheckHealth(ctx context.Context) error {
	return nil
}

func (n *EspressoFinalityNode) Initialize(ctx context.Context) error {
	return nil
}
