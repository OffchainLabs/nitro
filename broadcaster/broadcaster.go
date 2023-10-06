// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcaster

import (
	"context"
	"net"

	"github.com/gobwas/ws"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/http/backlog"
	httpServer "github.com/offchainlabs/nitro/broadcaster/http/server"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Broadcaster struct {
	server        *wsbroadcastserver.WSBroadcastServer
	catchupBuffer *SequenceNumberCatchupBuffer
	httpBacklog   backlog.Backlog
	chainId       uint64
	dataSigner    signature.DataSignerFunc
	httpServer    *httpServer.HTTPBroadcastServer
}

func NewBroadcaster(config wsbroadcastserver.BroadcasterConfigFetcher, chainId uint64, feedErrChan chan error, dataSigner signature.DataSignerFunc) *Broadcaster {
	catchupBuffer := NewSequenceNumberCatchupBuffer(func() bool { return config().LimitCatchup }, func() int { return config().MaxCatchup })
	httpBacklog := backlog.NewBacklog(func() *backlog.Config { return &config().HTTP.Backlog })
	return &Broadcaster{
		server:        wsbroadcastserver.NewWSBroadcastServer(config, catchupBuffer, chainId, feedErrChan),
		catchupBuffer: catchupBuffer,
		httpBacklog:   httpBacklog,
		chainId:       chainId,
		dataSigner:    dataSigner,
		httpServer:    httpServer.NewHTTPBroadcastServer(func() *httpServer.Config { return &config().HTTP }, httpBacklog),
	}
}

func (b *Broadcaster) NewBroadcastFeedMessage(message arbostypes.MessageWithMetadata, sequenceNumber arbutil.MessageIndex) (*m.BroadcastFeedMessage, error) {
	var messageSignature []byte
	if b.dataSigner != nil {
		hash, err := message.Hash(sequenceNumber, b.chainId)
		if err != nil {
			return nil, err
		}
		messageSignature, err = b.dataSigner(hash.Bytes())
		if err != nil {
			return nil, err
		}
	}

	return &m.BroadcastFeedMessage{
		SequenceNumber: sequenceNumber,
		Message:        message,
		Signature:      messageSignature,
	}, nil
}

func (b *Broadcaster) BroadcastSingle(msg arbostypes.MessageWithMetadata, seq arbutil.MessageIndex) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("recovered error in BroadcastSingle", "recover", r)
		}
	}()
	bfm, err := b.NewBroadcastFeedMessage(msg, seq)
	if err != nil {
		return err
	}

	b.BroadcastSingleFeedMessage(bfm)
	return nil
}

func (b *Broadcaster) BroadcastSingleFeedMessage(bfm *m.BroadcastFeedMessage) {
	broadcastFeedMessages := make([]*m.BroadcastFeedMessage, 0, 1)

	broadcastFeedMessages = append(broadcastFeedMessages, bfm)

	b.BroadcastFeedMessages(broadcastFeedMessages)
}

func (b *Broadcaster) BroadcastFeedMessages(messages []*m.BroadcastFeedMessage) {

	bm := m.BroadcastMessage{
		Version:  1,
		Messages: messages,
	}

	b.server.Broadcast(bm)
	if err := b.httpBacklog.Append(&bm); err != nil {
		log.Error("error whilst appending to HTTP backlog", "err", err)
	}
}

func (b *Broadcaster) Confirm(seq arbutil.MessageIndex) {
	log.Debug("confirming sequence number", "sequenceNumber", seq)
	bm := m.BroadcastMessage{
		Version: 1,
		ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{
			SequenceNumber: seq,
		},
	}
	b.server.Broadcast(bm)
	if err := b.httpBacklog.Append(&bm); err != nil {
		log.Error("error whilst appending to HTTP backlog", "err", err)
	}
}

func (b *Broadcaster) ClientCount() int32 {
	return b.server.ClientCount()
}

func (b *Broadcaster) ListenerAddr() net.Addr {
	return b.server.ListenerAddr()
}

func (b *Broadcaster) HTTPAddr() net.Addr {
	return b.httpServer.Addr()
}

func (b *Broadcaster) GetCachedMessageCount() int {
	return b.catchupBuffer.GetMessageCount()
}

func (b *Broadcaster) HTTPBacklogMessageCount() int {
	return b.httpBacklog.MessageCount()
}

func (b *Broadcaster) Initialize() error {
	return b.server.Initialize()
}

func (b *Broadcaster) Start(ctx context.Context) error {
	err := b.httpServer.Start()
	if err != nil {
		return err
	}
	return b.server.Start(ctx)
}

func (b *Broadcaster) StartWithHeader(ctx context.Context, header ws.HandshakeHeader) error {
	return b.server.StartWithHeader(ctx, header)
}

func (b *Broadcaster) StopAndWait() {
	b.server.StopAndWait()
	err := b.httpServer.StopAndWait()
	if err != nil {
		// Need to handle these errors better, should probably return errors up the stack
		log.Error(err.Error())
	}
}

func (b *Broadcaster) Started() bool {
	return b.server.Started()
}
