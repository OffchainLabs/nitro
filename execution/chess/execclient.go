package chess

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ChessNode struct {
	stopwaiter.StopWaiter
	past    map[arbutil.MessageIndex]*execution.MessageResult
	headMsg arbutil.MessageIndex
	engine  *ChessEngine
}

func NewChessNode(engine *ChessEngine) *ChessNode {
	return &ChessNode{
		engine: engine,
		past:   make(map[arbutil.MessageIndex]*execution.MessageResult),
	}
}

func (n *ChessNode) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	if n.headMsg+1 != msgIdx {
		return containers.NewReadyPromise(&execution.MessageResult{}, fmt.Errorf("digest message %d but we are at %d", msgIdx, n.headMsg))
	}
	err := n.engine.Process(msg.Message.Header.Poster, msg.Message.L2msg)
	if err != nil {
		log.Info("failed to process chess tx", "err", err, "msgIdx", msgIdx, "l2msg", msg.Message.L2msg)
	}
	result := execution.MessageResult{
		BlockHash: common.Hash(crypto.Keccak256([]byte(n.engine.Status()))),
	}
	n.past[msgIdx] = &result
	n.headMsg += 1
	return containers.NewReadyPromise(&result, nil)
}

func (n *ChessNode) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	return containers.NewReadyPromise[[]*execution.MessageResult](nil, errors.New("reorg not supported"))
}

func (n *ChessNode) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise(n.headMsg, nil)
}

func (n *ChessNode) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	result, exists := n.past[msgIdx]
	if !exists {
		containers.NewReadyPromise(&result, fmt.Errorf("result does not exist for: %d", msgIdx))
	}
	return containers.NewReadyPromise(result, nil)

}

func (n *ChessNode) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise(uint64(messageNum), nil)
}

func (n *ChessNode) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise(arbutil.MessageIndex(blockNum), nil)
}

func (n *ChessNode) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (n *ChessNode) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (n *ChessNode) Maintenance() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (n *ChessNode) Start(ctx context.Context) containers.PromiseInterface[struct{}] {
	n.StopWaiter.Start(ctx, n)
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (n *ChessNode) StopAndWait() containers.PromiseInterface[struct{}] {
	n.StopWaiter.StopAndWait()
	return containers.NewReadyPromise(struct{}{}, nil)
}
