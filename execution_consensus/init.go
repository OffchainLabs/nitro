package execution_consensus

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func InitAndStartExecutionAndConsensusNodes(ctx context.Context, stack *node.Node, execNode *gethexec.ExecutionNode, consensusNode *arbnode.Node) (func(), error) {
	if execNode != nil {
		if err := execNode.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("error initializing exec node: %w", err)
		}
	}
	if err := stack.Start(); err != nil {
		return nil, fmt.Errorf("error starting geth stack: %w", err)
	}
	if execNode != nil {
		execNode.SetConsensusClient(consensusNode)
		if err := execNode.Start(ctx); err != nil {
			return nil, fmt.Errorf("error starting exec node: %w", err)
		}
	}
	if err := consensusNode.Start(ctx); err != nil {
		return nil, fmt.Errorf("error starting consensus node: %w", err)
	}
	return func() {
		consensusNode.StopAndWait()
		if execNode != nil {
			execNode.StopAndWait()
		}
		if err := stack.Close(); err != nil {
			log.Error("error on stack close", "err", err)
		}
	}, nil
}
