// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package data_streaming

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const (
	DefaultMaxPendingMessages      = 10
	DefaultMessageCollectionExpiry = 1 * time.Minute
	DefaultRequestValidity         = 5 * time.Minute
)

// DataStreamReceiver implements the server side of the data streaming protocol. It stays compatible with `DataStreamer`
// client, although is able to talk to many senders at the same time.
//
// DataStreamReceiver is responsible only for the protocol level communication. Usually it will be wrapped within an
// outer service that handles JSON-serialization, HTTP, listening etc.
//
// DataStreamReceiver has built-in support for limiting number of open protocol interactions and garbage collection for
// the interrupted streams.
// lint:require-exhaustive-initialization
type DataStreamReceiver struct {
	stopwaiter.StopWaiter

	payloadVerifier *PayloadVerifier
	messageStore    *messageStore
	requestValidity time.Duration

	mutex        sync.Mutex
	seenRequests map[common.Hash]time.Time
}

func (dsr *DataStreamReceiver) Start(ctxIn context.Context) {
	dsr.StopWaiter.Start(ctxIn, dsr)

	dsr.StopWaiter.CallIteratively(func(ctx context.Context) time.Duration {
		dsr.mutex.Lock()
		defer dsr.mutex.Unlock()
		cutoff := time.Now().Add(-dsr.requestValidity)
		for hash, requestTime := range dsr.seenRequests {
			if requestTime.Before(cutoff) {
				delete(dsr.seenRequests, hash)
			}
		}
		return dsr.requestValidity
	})
}

// NewDataStreamReceiver sets up a new stream receiver. `payloadVerifier` must be compatible with message signing on
// the `DataStreamer` sender side. `maxPendingMessages` limits how many parallel protocol instances are supported.
// `messageCollectionExpiry` is the window in which a protocol must end - otherwise the protocol will be closed and all
// related data will be removed. This time window is reset after every _new_ protocol message received.
// `requestValidity` is the maximum age of the incoming protocol opening message.
func NewDataStreamReceiver(payloadVerifier *PayloadVerifier, maxPendingMessages int, messageCollectionExpiry, requestValidity time.Duration, expirationCallback func(id MessageId)) *DataStreamReceiver {
	return &DataStreamReceiver{
		StopWaiter: stopwaiter.StopWaiter{},

		payloadVerifier: payloadVerifier,
		messageStore:    newMessageStore(maxPendingMessages, messageCollectionExpiry, expirationCallback),
		requestValidity: requestValidity,

		mutex:        sync.Mutex{},
		seenRequests: make(map[common.Hash]time.Time),
	}
}

// NewDefaultDataStreamReceiver sets up a new stream receiver with default settings.
func NewDefaultDataStreamReceiver(verifier *PayloadVerifier) *DataStreamReceiver {
	return NewDataStreamReceiver(verifier, DefaultMaxPendingMessages, DefaultMessageCollectionExpiry, DefaultRequestValidity, nil)
}

// StartStreamingResult is expected by DataStreamer to be returned by the endpoint responsible for the StartReceiving method.
// lint:require-exhaustive-initialization
type StartStreamingResult struct {
	MessageId hexutil.Uint64 `json:"BatchId,omitempty"` // For compatibility reasons we keep the old name "BatchId"
}

func (dsr *DataStreamReceiver) StartReceiving(ctx context.Context, timestamp, nChunks, chunkSize, totalSize, timeout uint64, signature []byte) (*StartStreamingResult, error) {
	dsr.mutex.Lock()
	defer dsr.mutex.Unlock()

	if err := dsr.payloadVerifier.verifyPayload(ctx, signature, []byte{}, timestamp, nChunks, chunkSize, totalSize, timeout); err != nil {
		return &StartStreamingResult{0}, err
	}

	requestTime := time.Unix(int64(timestamp), 0) // #nosec G115

	// Deny too old or from-future requests. We keep the margin of `dsr.requestValidity` also to the future, in case of unsync clocks.
	if time.Since(requestTime).Abs() > dsr.requestValidity {
		return &StartStreamingResult{0}, errors.New("too much time has elapsed since request was signed")
	}

	// Save in a short-memory cache that request. We use the first 32 bytes of the signature as a sufficient, pseudorandom request identifier.
	requestHash := common.BytesToHash(signature)
	if _, ok := dsr.seenRequests[requestHash]; ok {
		return &StartStreamingResult{0}, errors.New("we have already seen this request; aborting replayed protocol")
	}
	dsr.seenRequests[requestHash] = requestTime

	messageId, err := dsr.messageStore.registerNewMessage(nChunks, timeout, chunkSize, totalSize)
	return &StartStreamingResult{hexutil.Uint64(messageId)}, err
}

func (dsr *DataStreamReceiver) ReceiveChunk(ctx context.Context, messageId MessageId, chunkId uint64, chunkData, signature []byte) error {
	if err := dsr.payloadVerifier.verifyPayload(ctx, signature, chunkData, uint64(messageId), chunkId); err != nil {
		return err
	}
	return dsr.messageStore.addNewChunk(messageId, chunkId, chunkData)
}

func (dsr *DataStreamReceiver) FinalizeReceiving(ctx context.Context, messageId MessageId, signature hexutil.Bytes) ([]byte, uint64, time.Time, error) {
	if err := dsr.payloadVerifier.verifyPayload(ctx, signature, []byte{}, uint64(messageId)); err != nil {
		return nil, 0, time.Time{}, err
	}
	return dsr.messageStore.finalizeMessage(messageId)
}

// ============= MESSAGE MANAGEMENT ================================================================================= //

// MessageId is the identifier of the message being streamed (protocol invocation id).
type MessageId uint64

// lint:require-exhaustive-initialization
type partialMessage struct {
	mutex             sync.Mutex
	chunks            [][]byte
	seenChunks        int
	expectedChunkSize uint64
	expectedTotalSize uint64
	timeout           uint64
	startTime         time.Time
	lastUpdateTime    time.Time
}

// lint:require-exhaustive-initialization
type messageStore struct {
	mutex                   sync.Mutex
	messages                map[MessageId]*partialMessage
	maxPendingMessages      int
	messageCollectionExpiry time.Duration
	expirationCallback      func(MessageId)
}

func newMessageStore(maxPendingMessages int, messageCollectionExpiry time.Duration, expirationCallback func(id MessageId)) *messageStore {
	return &messageStore{
		mutex:                   sync.Mutex{},
		messages:                make(map[MessageId]*partialMessage),
		maxPendingMessages:      maxPendingMessages,
		messageCollectionExpiry: messageCollectionExpiry,
		expirationCallback:      expirationCallback,
	}
}

func (ms *messageStore) registerNewMessage(nChunks, timeout, chunkSize, totalSize uint64) (id MessageId, err error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Validate parameters to prevent inconsistent or unsafe states.
	if nChunks*totalSize*chunkSize == 0 {
		return 0, errors.New("can't start collecting new message: neither number of chunks, total size or chunk size can be zero")
	}
	if len(ms.messages) >= ms.maxPendingMessages {
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
		mutex:             sync.Mutex{},
		chunks:            make([][]byte, nChunks),
		seenChunks:        0,
		expectedChunkSize: chunkSize,
		expectedTotalSize: totalSize,
		timeout:           timeout,
		startTime:         time.Now(),
		lastUpdateTime:    time.Now(),
	}

	// Schedule garbage collection for the old incomplete messages.
	var gcRoutine func()
	gcRoutine = func() {
		ms.mutex.Lock()
		defer ms.mutex.Unlock()

		message, stillExists := ms.messages[id]
		if !stillExists {
			return
		} else if time.Since(message.lastUpdateTime) > ms.messageCollectionExpiry {
			if ms.expirationCallback != nil {
				ms.expirationCallback(id)
			}
			delete(ms.messages, id)
			return
		}
		time.AfterFunc(ms.messageCollectionExpiry, gcRoutine)
	}
	go gcRoutine()

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
		if bytes.Equal(message.chunks[chunkId], chunk) {
			// Server idempotency: ignore duplicated request as long as it doesn't break consistency
			return nil
		} else {
			// Inconsistency between chunks at the same index detected. Protocol must be aborted (no way of deciding what data is correct)
			delete(ms.messages, id)
			return errors.New("received different chunk data than previously; aborting protocol")
		}
	}

	// Validate chunk size
	chunkLen := uint64(len(chunk))
	if chunkId+1 == uint64(len(message.chunks)) {
		// For the last chunk, if totalSize is an exact multiple of chunkSize,
		// the expected size is chunkSize (not zero remainder).
		expectedLen := (message.expectedTotalSize-1)%message.expectedChunkSize + 1
		if chunkLen != expectedLen {
			return fmt.Errorf("message(%d): chunk(%d) has incorrect size (%d bytes) - expecting %d bytes", id, chunkId, chunkLen, expectedLen)
		}
	} else if chunkLen != message.expectedChunkSize {
		return fmt.Errorf("message(%d): chunk(%d) has incorrect size (%d bytes) - expecting %d bytes", id, chunkId, chunkLen, message.expectedChunkSize)
	}

	message.chunks[chunkId] = chunk
	message.seenChunks++
	message.lastUpdateTime = time.Now()

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
