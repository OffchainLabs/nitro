package nethexec

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	_ FullExecutionClient         = (*nethermindExecutionClient)(nil)
	_ arbnode.ExecutionNodeBridge = (*nethermindExecutionClient)(nil)
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

func (p *nethermindExecutionClient) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)
	go func() {
		res := p.rpcClient.DigestMessage(context.Background(), num, msg, msgForPrefetch)
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
		idx, err := p.rpcClient.HeadMessageNumber(context.Background())
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
		res, err := p.rpcClient.ResultAtPos(context.Background(), msgIdx)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

func (p *nethermindExecutionClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	promise := containers.NewPromise[uint64](nil)
	go func() {
		num, err := p.rpcClient.MessageIndexToBlockNumber(context.Background(), messageNum)
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

func (p *nethermindExecutionClient) Maintenance() containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		promise.ProduceError(fmt.Errorf("Maintenance not implemented"))
	}()
	return &promise
}

func (p *nethermindExecutionClient) Start(ctx context.Context) error {
	return fmt.Errorf("Start not implemented")
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

func (p *nethermindExecutionClient) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	return nil, fmt.Errorf("RecordBlockCreation not implemented")
}

func (p *nethermindExecutionClient) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
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

func (w *nethermindExecutionClient) Initialize(ctx context.Context) error {
	return fmt.Errorf("Initialize not implemented")
}
