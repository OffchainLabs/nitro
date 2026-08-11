// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package melrpcserver is the node-side server for all provider->node MEL RPC, exposed under
// the "nitromelconsumer" JSON-RPC namespace. It runs on the Nitro node; a remote MEL provider
// connects as an RPC client and:
//   - pushes the messages it extracts (PushMessages -> the node's TransactionStreamer), and
//   - notifies the node when it reorgs (ReorgedToParentChainBlock -> the node's melReorgDetector,
//     which local consumers read to rewind).
//
// This is the producer->consumer direction, mirroring how the consensus client pushes into the
// execution server. Note: a provider that receives a node->provider ReorgTo (meldataprovider
// namespace) must call back ReorgedToParentChainBlock here so local consumers rewind.
package melrpcserver

import (
	"context"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// Server wraps the local message consumer (TransactionStreamer) and the local reorg-notifier
// channel (melReorgDetector), exposing them under mel.ConsumerRPCNamespace ("nitromelconsumer").
type Server struct {
	consumer      mel.MessageConsumer
	reorgNotifier chan<- uint64
}

// NewServer builds the node-side MEL RPC server. reorgNotifier is the node's melReorgDetector
// channel, or nil when this node has nothing to rewind (reorg notifications are then ignored).
func NewServer(consumer mel.MessageConsumer, reorgNotifier chan<- uint64) *Server {
	return &Server{consumer: consumer, reorgNotifier: reorgNotifier}
}

// PushMessages forwards extracted messages to the local consumer. The consumer
// (TransactionStreamer) is position-keyed and reorg-safe, so overlapping/re-pushed
// ranges and parent-chain reorgs are handled there.
func (s *Server) PushMessages(ctx context.Context, firstMsgIdx uint64, messages []*arbostypes.MessageWithMetadata) error {
	return s.consumer.PushMessages(ctx, firstMsgIdx, messages)
}

// ReorgedToParentChainBlock is called by the remote MEL provider when it reorgs; it feeds the
// local melReorgDetector so the node's reorg consumers rewind. It is a no-op if this node has
// no consumer (nil channel). The send is ctx-guarded so a shutdown/timeout cannot wedge the
// handler; melReorgDetector is buffered on the node side to tolerate the reader starting after
// the RPC endpoint goes live.
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
