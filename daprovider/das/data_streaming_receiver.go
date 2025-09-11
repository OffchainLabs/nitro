// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type DataStreamReceiver struct {
	signatureVerifier *SignatureVerifier
	messageStore      *messageStore
}

func NewDataStreamReceiver(signatureVerifier *SignatureVerifier) *DataStreamReceiver {
	return &DataStreamReceiver{
		signatureVerifier: signatureVerifier,
		messageStore:      newMessageStore(),
	}
}

func (dsr *DataStreamReceiver) StartReceiving(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout uint64, sig []byte) (MessageId, error) {
	if err := dsr.signatureVerifier.verify(ctx, []byte{}, sig, timestamp, nChunks, chunkSize, totalSize, timeout); err != nil {
		return 0, err
	}

	// Prevent replay of old messages
	// #nosec G115
	if time.Since(time.Unix(int64(timestamp), 0)).Abs() > time.Minute {
		return 0, errors.New("too much time has elapsed since request was signed")
	}

	return dsr.messageStore.registerNewMessage(nChunks, timeout, chunkSize, totalSize)
}

func (dsr *DataStreamReceiver) ReceiveChunk(ctx context.Context, messageId MessageId, chunkId uint64, chunk, sig []byte) error {
	if err := dsr.signatureVerifier.verify(ctx, chunk, sig, uint64(messageId), chunkId); err != nil {
		return err
	}
	return dsr.messageStore.addNewChunk(messageId, chunkId, chunk)
}

func (dsr *DataStreamReceiver) FinalizeReceiving(ctx context.Context, messageId MessageId, sig hexutil.Bytes) ([]byte, uint64, time.Time, error) {
	if err := dsr.signatureVerifier.verify(ctx, []byte{}, sig, uint64(messageId)); err != nil {
		return nil, 0, time.Time{}, err
	}
	return dsr.messageStore.finalizeMessage(messageId)
}

// ============= MESSAGE MANAGEMENT ================================================================================= //

type MessageId uint64

// todo remove
const (
	maxPendingMessages      = 10
	messageCollectionExpiry = 1 * time.Minute
)

type partialMessage struct {
	mutex             sync.Mutex
	chunks            [][]byte
	seenChunks        int
	expectedChunkSize uint64
	expectedTotalSize uint64
	timeout           uint64
	startTime         time.Time
}

type messageStore struct {
	mutex    sync.Mutex
	messages map[MessageId]*partialMessage
}

func newMessageStore() *messageStore {
	return &messageStore{
		mutex:    sync.Mutex{},
		messages: make(map[MessageId]*partialMessage),
	}
}

func (ms *messageStore) registerNewMessage(nChunks, timeout, chunkSize, totalSize uint64) (id MessageId, err error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	if len(ms.messages) >= maxPendingMessages {
		return 0, fmt.Errorf("can't start collecting new message: already %d pending", len(ms.messages))
	}

	// Find a free identifier.
	for {
		id = MessageId(rand.Uint64())
		if _, alreadyRegistered := ms.messages[id]; !alreadyRegistered {
			break
		}
	}

	ms.messages[id] = &partialMessage{
		chunks:            make([][]byte, nChunks),
		expectedChunkSize: chunkSize,
		expectedTotalSize: totalSize,
		timeout:           timeout,
		startTime:         time.Now(),
	}

	// Schedule garbage collection for the old incomplete messages.
	go func(id MessageId) {
		<-time.After(messageCollectionExpiry)
		ms.mutex.Lock()
		defer ms.mutex.Unlock()

		// Message will only exist if expiry was reached without it being complete.
		if _, stillExists := ms.messages[id]; stillExists {
			rpcStoreFailureGauge.Inc(1)
			delete(ms.messages, id)
		}
	}(id)

	return id, nil
}

func (ms *messageStore) addNewChunk(id MessageId, chunkId uint64, chunk []byte) error {
	ms.mutex.Lock()
	message, ok := ms.messages[id]
	ms.mutex.Unlock()

	if !ok {
		return fmt.Errorf("unknown message(%d)", id)
	}

	message.mutex.Lock()
	defer message.mutex.Unlock()

	if chunkId >= uint64(len(message.chunks)) {
		return fmt.Errorf("message(%d): chunk(%d) out of range - expected %d chunks", id, chunkId, len(message.chunks))
	}

	if message.chunks[chunkId] != nil {
		return fmt.Errorf("message(%d): chunk(%d) already added", id, chunkId)
	}

	// Validate chunk size
	chunkLen := uint64(len(chunk))
	if chunkId+1 == uint64(len(message.chunks)) {
		expectedLen := message.expectedTotalSize % message.expectedChunkSize
		if chunkLen != expectedLen {
			return fmt.Errorf("message(%d): chunk(%d) has incorrect size (%d bytes) - expecting %d bytes", id, chunkId, chunkLen, expectedLen)
		}
	} else if chunkLen != message.expectedChunkSize {
		return fmt.Errorf("message(%d): chunk(%d) has incorrect size (%d bytes) - expecting %d bytes", id, chunkId, chunkLen, message.expectedChunkSize)
	}

	message.chunks[chunkId] = chunk
	message.seenChunks++

	return nil
}

func (ms *messageStore) finalizeMessage(id MessageId) ([]byte, uint64, time.Time, error) {
	ms.mutex.Lock()
	message, messageIsRegistered := ms.messages[id]
	delete(ms.messages, id)
	ms.mutex.Unlock()
	if !messageIsRegistered {
		return nil, 0, time.Time{}, fmt.Errorf("unknown message(%d)", id)
	}

	message.mutex.Lock()
	defer message.mutex.Unlock()

	if len(message.chunks) != message.seenChunks {
		return nil, 0, time.Time{}, fmt.Errorf("incomplete message(%d): got %d/%d chunks", id, message.seenChunks, len(message.chunks))
	}

	var flattened []byte
	for _, chunk := range message.chunks {
		flattened = append(flattened, chunk...)
	}

	return flattened, message.timeout, message.startTime, nil
}
