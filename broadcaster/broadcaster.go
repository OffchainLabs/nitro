// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcaster

import (
	"context"
	"errors"
	"net"
	"runtime/debug"

	"github.com/gobwas/ws"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

const SupportedBroadcastVersion = m.V1

type Broadcaster struct {
	server     *wsbroadcastserver.WSBroadcastServer
	backlog    backlog.Backlog
	chainId    uint64
	dataSigner signature.DataSignerFunc
}

func NewBroadcaster(config wsbroadcastserver.BroadcasterConfigFetcher, chainId uint64, feedErrChan chan error, dataSigner signature.DataSignerFunc) *Broadcaster {
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config().Backlog })
	return &Broadcaster{
		server:     wsbroadcastserver.NewWSBroadcastServer(config, bklg, chainId, feedErrChan),
		backlog:    bklg,
		chainId:    chainId,
		dataSigner: dataSigner,
	}
}

func (b *Broadcaster) NewBroadcastFeedMessage(
	message arbostypes.MessageWithMetadataAndBlockInfo,
	sequenceNumber arbutil.MessageIndex,
) (*m.BroadcastFeedMessage, error) {
	feedMessage := m.BroadcastFeedMessage{
		SequenceNumber: sequenceNumber,
		Message:        message.MessageWithMeta,
		BlockHash:      message.BlockHash,
		Signature:      []byte{},
		BlockMetadata:  message.BlockMetadata,
	}
	if b.dataSigner != nil {
		hash := feedMessage.SignatureHash(b.chainId)
		var err error
		feedMessage.Signature, err = b.dataSigner(hash.Bytes())
		if err != nil {
			return nil, err
		}
	}
	return &feedMessage, nil
}

func (b *Broadcaster) BroadcastFeedMessages(messages []*m.BroadcastFeedMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("recovered error in BroadcastFeedMessages", "recover", r, "backtrace", string(debug.Stack()))
			err = errors.New("panic in BroadcastSingle")
		}
	}()

	bm := &m.BroadcastMessage{
		Version:  SupportedBroadcastVersion,
		Messages: messages,
	}
	b.server.Broadcast(bm)
	return
}

func (b *Broadcaster) PopulateFeedBacklog(messages []*m.BroadcastFeedMessage) error {
	bm := &m.BroadcastMessage{
		Version:  SupportedBroadcastVersion,
		Messages: messages,
	}
	return b.server.PopulateFeedBacklog(bm)
}

func (b *Broadcaster) Confirm(msgIdx arbutil.MessageIndex) {
	log.Debug("confirming msgIdx", "msgIdx", msgIdx)
	b.server.Broadcast(&m.BroadcastMessage{
		Version: SupportedBroadcastVersion,
		ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{
			SequenceNumber: msgIdx,
		},
	})
}

func (b *Broadcaster) ClientCount() int32 {
	return b.server.ClientCount()
}

func (b *Broadcaster) ListenerAddr() net.Addr {
	return b.server.ListenerAddr()
}

func (b *Broadcaster) GetCachedMessageCount() int {
	// #nosec G115
	return int(b.backlog.Count())
}

func (b *Broadcaster) Initialize() error {
	return b.server.Initialize()
}

func (b *Broadcaster) Start(ctx context.Context) error {
	return b.server.Start(ctx)
}

func (b *Broadcaster) StartWithHeader(ctx context.Context, header ws.HandshakeHeader) error {
	return b.server.StartWithHeader(ctx, header)
}

func (b *Broadcaster) StopAndWait() {
	b.server.StopAndWait()
}

func (b *Broadcaster) Started() bool {
	return b.server.Started()
}
