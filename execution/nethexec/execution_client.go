package nethexec

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	_ FullExecutionClient         = (*nethermindExecutionClient)(nil)
	_ arbnode.ExecutionNodeBridge = (*nethermindExecutionClient)(nil)
	_ InitMessageDigester         = (*nethermindExecutionClient)(nil)
)

type nethermindExecutionClient struct {
	rpcClient *nethRpcClient
}

func NewNethermindExecutionClient() (*nethermindExecutionClient, error) {
	rpcClient, err := NewNethRpcClient()
	if err != nil {
		return nil, err
	}
	return &nethermindExecutionClient{
		rpcClient: rpcClient,
	}, nil
}

func (p *nethermindExecutionClient) DigestMessage(index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)
	go func() {
		res := p.rpcClient.DigestMessage(context.Background(), index, msg, msgForPrefetch)
		if res == nil {
			promise.ProduceError(fmt.Errorf("external DigestMessage returned nil"))
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

func (p *nethermindExecutionClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		err := p.rpcClient.SetFinalityData(ctx, safeFinalityData, finalizedFinalityData, validatedFinalityData)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	}()
	return &promise
}
func (p *nethermindExecutionClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	promise := containers.NewPromise[[]*execution.MessageResult](nil)
	go func() {
		res, err := p.rpcClient.Reorg(context.Background(), msgIdxOfFirstMsgToAdd, newMessages, oldMessages)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

func (p *nethermindExecutionClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	promise := containers.NewPromise[arbutil.MessageIndex](nil)
	go func() {
		idx, err := p.rpcClient.HeadMessageIndex(context.Background())
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(idx)
	}()
	return &promise
}

func (p *nethermindExecutionClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)
	go func() {
		res, err := p.rpcClient.ResultAtMessageIndex(context.Background(), msgIdx)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

func (p *nethermindExecutionClient) MessageIndexToBlockNumber(messageIndex arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	promise := containers.NewPromise[uint64](nil)
	go func() {
		num, err := p.rpcClient.MessageIndexToBlockNumber(context.Background(), messageIndex)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(num)
	}()
	return &promise
}

func (p *nethermindExecutionClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	promise := containers.NewPromise[arbutil.MessageIndex](nil)
	go func() {
		idx, err := p.rpcClient.BlockNumberToMessageIndex(context.Background(), blockNum)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(idx)
	}()
	return &promise
}

func (p *nethermindExecutionClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		err := p.rpcClient.MarkFeedStart(context.Background(), to)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	}()
	return &promise
}

func (p *nethermindExecutionClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		promise.ProduceError(fmt.Errorf("TriggerMaintenance not implemented"))
	}()
	return &promise
}

func (p *nethermindExecutionClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	promise := containers.NewPromise[bool](nil)
	go func() {
		// Conservative default - don't trigger maintenance until implemented
		promise.Produce(false)
	}()
	return &promise
}

func (p *nethermindExecutionClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	promise := containers.NewPromise[*execution.MaintenanceStatus](nil)
	go func() {
		// Return default status indicating maintenance is not running
		status := &execution.MaintenanceStatus{
			IsRunning: false,
		}
		promise.Produce(status)
	}()
	return &promise
}

func (p *nethermindExecutionClient) Start(ctx context.Context) error {
	if p.rpcClient == nil {
		return fmt.Errorf("RPC client is not initialized")
	}

	// TODO: Add a health check RPC call to verify Nethermind is accessible
	// For now, we'll return success to allow the test to proceed
	return nil
}

func (p *nethermindExecutionClient) StopAndWait() {
	// no-op default until implemented
}

func (p *nethermindExecutionClient) Pause() {
	// no-op default until implemented
}

func (p *nethermindExecutionClient) Activate() {
	// no-op default until implemented
}

func (p *nethermindExecutionClient) ForwardTo(url string) error {
	return fmt.Errorf("ForwardTo not implemented")
}

func (p *nethermindExecutionClient) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	return p.rpcClient.SequenceDelayedMessage(context.Background(), message, delayedSeqNum)
}

func (p *nethermindExecutionClient) NextDelayedMessageNumber() (uint64, error) {
	return 0, fmt.Errorf("NextDelayedMessageNumber not implemented")
}

func (p *nethermindExecutionClient) Synced(ctx context.Context) bool {
	// default conservative value until implemented
	return false
}

func (p *nethermindExecutionClient) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *nethermindExecutionClient) RecordBlockCreation(ctx context.Context, index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	return nil, fmt.Errorf("RecordBlockCreation not implemented")
}

func (p *nethermindExecutionClient) MarkValid(index arbutil.MessageIndex, resultHash common.Hash) {
	// no-op until implemented
}

func (p *nethermindExecutionClient) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	return fmt.Errorf("PrepareForRecord not implemented")
}

func (p *nethermindExecutionClient) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error) {
	return 0, fmt.Errorf("ArbOSVersionForMessageIndex not implemented")
}

func (w *nethermindExecutionClient) SetConsensusClient(consensus execution.FullConsensusClient) {
	// no-op until consensus path is implemented
}

func (w *nethermindExecutionClient) DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult {
	return w.rpcClient.DigestInitMessage(ctx, initialL1BaseFee, serializedChainConfig)
}

func (w *nethermindExecutionClient) Initialize(ctx context.Context) error {
	if w.rpcClient == nil {
		return fmt.Errorf("RPC client is not initialized")
	}

	// TODO: Add a health check RPC call to verify Nethermind is accessible
	// For now, we'll return success to allow the test to proceed
	return nil
}

func (p *nethermindExecutionClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	result, err := p.rpcClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ethereum.NotFound
	}

	// Type assertion to convert interface{} to *types.Receipt
	receipt, ok := result.(*types.Receipt)
	if !ok {
		return nil, fmt.Errorf("unexpected receipt type: %T", result)
	}
	return receipt, nil
}

func (p *nethermindExecutionClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (*rpc.ClientSubscription, error) {
	return p.rpcClient.SubscribeNewHead(ctx)
}

func (p *nethermindExecutionClient) BlockNumber(ctx context.Context) (uint64, error) {
	return p.rpcClient.BlockNumber(ctx)
}

func (p *nethermindExecutionClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return p.rpcClient.BalanceAt(ctx, account, blockNumber)
}
