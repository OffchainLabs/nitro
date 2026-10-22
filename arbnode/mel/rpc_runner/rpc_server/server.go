// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcserver serves the provider->node side of MEL RPC on the Nitro node, under the
// "nitromelconsumer" namespace: a remote provider connects as a client and pushes the messages it
// extracts plus its reorg notifications. Mirrors how the consensus client pushes into execution.
package melrpcserver

import (
	"context"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// Server exposes the local message consumer and reorg-notifier channel over RPC.
type Server struct {
	consumer      mel.MessageConsumer
	reorgNotifier chan<- uint64
}

// NewServer takes the node's melReorgDetector as reorgNotifier, or nil to ignore reorgs.
func NewServer(consumer mel.MessageConsumer, reorgNotifier chan<- uint64) *Server {
	return &Server{consumer: consumer, reorgNotifier: reorgNotifier}
}

// PushMessages forwards to the consumer, which is position-keyed and reorg-safe, so re-pushed
// ranges are handled there.
func (s *Server) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	return s.consumer.PushMessages(ctx, firstMsgIdx, messages)
}

// ReorgedToParentChainBlock feeds melReorgDetector so local consumers rewind. The send is
// ctx-guarded so a shutdown cannot wedge the handler.
func (s *Server) ReorgedToParentChainBlock(ctx context.Context, parentChainBlockNumber uint64) error {
	if s.reorgNotifier == nil {
		return nil // nothing on this node consumes MEL reorg notifications
	}
	select {
	case s.reorgNotifier <- parentChainBlockNumber:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
