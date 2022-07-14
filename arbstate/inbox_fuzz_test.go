// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"context"
	"errors"
	"testing"
)

type multiplexerBackend struct {
	batchSeqNum           uint64
	batch                 []byte
	delayedMessage        []byte
	positionWithinMessage uint64
}

func (b *multiplexerBackend) PeekSequencerInbox() ([]byte, error) {
	if b.batchSeqNum != 0 {
		return nil, errors.New("reading unknown sequencer batch")
	}
	return b.batch, nil
}

func (b *multiplexerBackend) GetSequencerInboxPosition() uint64 {
	return b.batchSeqNum
}

func (b *multiplexerBackend) AdvanceSequencerInbox() {
	b.batchSeqNum++
}

func (b *multiplexerBackend) GetPositionWithinMessage() uint64 {
	return b.positionWithinMessage
}

func (b *multiplexerBackend) SetPositionWithinMessage(pos uint64) {
	b.positionWithinMessage = pos
}

func (b *multiplexerBackend) ReadDelayedInbox(seqNum uint64) ([]byte, error) {
	if seqNum != 0 {
		return nil, errors.New("reading unknown delayed message")
	}
	return b.delayedMessage, nil
}

func FuzzInboxMultiplexer(f *testing.F) {
	f.Fuzz(func(t *testing.T, seqMsg []byte, delayedMsg []byte) {
		if len(seqMsg) < 40 {
			return
		}
		backend := &multiplexerBackend{
			batchSeqNum:           0,
			batch:                 seqMsg,
			delayedMessage:        delayedMsg,
			positionWithinMessage: 0,
		}
		multiplexer := NewInboxMultiplexer(backend, 0, nil)
		_, err := multiplexer.Pop(context.TODO())
		if err != nil {
			panic(err)
		}
	})
}
