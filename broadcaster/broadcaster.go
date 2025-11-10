// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcaster

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/backlog"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Broadcaster struct {
	*wsbroadcastserver.WSBroadcastServer
	backlog    backlog.Backlog
	chainId    uint64
	dataSigner signature.DataSignerFunc
}

func NewBroadcaster(config wsbroadcastserver.BroadcasterConfigFetcher, chainId uint64, feedErrChan chan error, dataSigner signature.DataSignerFunc) *Broadcaster {
	bklg := backlog.NewBacklog(func() *backlog.Config { return &config().Backlog })
	return &Broadcaster{
		wsbroadcastserver.NewWSBroadcastServer(config, bklg, chainId, feedErrChan),
		bklg,
		chainId,
		dataSigner,
	}
}

func (b *Broadcaster) NewBroadcastFeedMessage(
	message arbostypes.MessageWithMetadataAndBlockInfo,
	sequenceNumber arbutil.MessageIndex,
) (*m.BroadcastFeedMessage, error) {
	var messageSignature []byte
	if b.dataSigner != nil {
		hash, err := message.MessageWithMeta.Hash(sequenceNumber, b.chainId)
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
		Message:        message.MessageWithMeta,
		BlockHash:      message.BlockHash,
		Signature:      messageSignature,
		BlockMetadata:  message.BlockMetadata,
	}, nil
}

func (b *Broadcaster) BroadcastFeedMessages(messages []*m.BroadcastFeedMessage) {
	bm := &m.BroadcastMessage{
		Version:  m.V1,
		Messages: messages,
	}
	b.Broadcast(bm)
}

func (b *Broadcaster) PopulateBacklog(messages []*m.BroadcastFeedMessage) error {
	bm := &m.BroadcastMessage{
		Version:  m.V1,
		Messages: messages,
	}
	return b.PopulateFeedBacklog(bm)
}

func (b *Broadcaster) Confirm(msgIdx arbutil.MessageIndex) {
	log.Debug("confirming msgIdx", "msgIdx", msgIdx)
	b.Broadcast(&m.BroadcastMessage{
		Version: m.V1,
		ConfirmedSequenceNumberMessage: &m.ConfirmedSequenceNumberMessage{
			SequenceNumber: msgIdx,
		},
	})
}

func (b *Broadcaster) GetCachedMessageCount() int {
	// #nosec G115
	return int(b.backlog.Count())
}
