// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
	stopAll := func() {
		consensusNode.StopAndWait()
		if execNode != nil {
			execNode.StopAndWait()
		}
		if err := stack.Close(); err != nil {
			log.Error("error on stack close", "err", err)
		}
	}
	var err error
	stackStartFailed := false
	defer func() {
		if err == nil {
			return
		}
		log.Error("Error initializing and starting execution and consensus", "err", err)
		// A failed node.Node.Start has already closed every database opened
		// through the stack and left the node in the closed state. Stopping
		// the blockchain now would write the snapshot journal to a closed
		// database, which is a log.Crit, and stack.Close would only return
		// ErrNodeStopped, so there is nothing to unwind on that path.
		if stackStartFailed {
			return
		}
		stopAll()
	}()
	if execNode != nil {
		if err = execNode.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("error initializing exec node: %w", err)
		}
	}
	if err = stack.Start(); err != nil {
		stackStartFailed = true
		return nil, fmt.Errorf("error starting geth stack: %w", err)
	}
	if execNode != nil {
		execNode.SetConsensusClient(consensusNode)
		if err = execNode.Start(ctx); err != nil {
			return nil, fmt.Errorf("error starting exec node: %w", err)
		}
	}
	if err = consensusNode.Start(ctx); err != nil {
		return nil, fmt.Errorf("error starting consensus node: %w", err)
	}
	return stopAll, nil
}
