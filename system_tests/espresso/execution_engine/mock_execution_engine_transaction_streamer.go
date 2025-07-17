package execution_engine

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

// MockExecutionEngineForTransactionStreamer is a mock implementation of the
// execution.ExecutionSequencer interface.
//
// It implements a minimal set of the methods required.
// It's current implementation is focused on targeting the methods required
// for testing the TransactionStreamer and EspressoChain functionality.
//
// NOTE: This mock *should* be safe to use between threads.
// NOTE: The Results map is *MUST* be initialized before any methods are called,
// it is recommended to use the NewMockExecutionEngineForTransactionStreamer
// function to ensure that the Results map is initialized.
type MockExecutionEngineForTransactionStreamer struct {
	execution.ExecutionClient
	Lock    sync.RWMutex
	Latest  arbutil.MessageIndex
	Results map[arbutil.MessageIndex]*execution.MessageResult
	Hasher  MessageHasher
}

// Compile time check to ensure that MockExecutionEngine implements the
// execution.ExecutionSequencer interface.
var _ execution.ExecutionClient = &MockExecutionEngineForTransactionStreamer{}

// MockExecutionEngineForTransactionStreamerConfig holds the configuration
// options for the MockExecutionEngineForTransactionStreamer.
type MockExecutionEngineForTransactionStreamerConfig struct {
	Results map[arbutil.MessageIndex]*execution.MessageResult
	Hasher  MessageHasher
}

// MockExecutionEngineForTransactionStreamerConfigOption is a function that
// modifies the MockExecutionEngineForTransactionStreamerConfig. This allows
// for flexible configuration of the mock execution engine.
type MockExecutionEngineForTransactionStreamerConfigOption func(cfg *MockExecutionEngineForTransactionStreamerConfig)

// WithHasher allows the caller to override the default hasher used by the
// MockExecutionEngineForTransactionStreamer.
func WithHasher(hasher MessageHasher) MockExecutionEngineForTransactionStreamerConfigOption {
	return func(cfg *MockExecutionEngineForTransactionStreamerConfig) {
		cfg.Hasher = hasher
	}
}

// NewMockExecutionEngineForTransactionStreamer returns an implementation of
// execution.ExecutionSequencer that can ber used for testing the
// TransactionStreamer.
func NewMockExecutionEngineForTransactionStreamer(options ...MockExecutionEngineForTransactionStreamerConfigOption) execution.ExecutionClient {
	config := MockExecutionEngineForTransactionStreamerConfig{
		Results: make(map[arbutil.MessageIndex]*execution.MessageResult),
		Hasher:  DefaultMessageHasher,
	}

	for _, option := range options {
		option(&config)
	}

	return &MockExecutionEngineForTransactionStreamer{
		Results: config.Results,
		Hasher:  config.Hasher,
	}
}

// DigestMessage implements execution.ExecutionSequencer.
func (m *MockExecutionEngineForTransactionStreamer) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	hash := m.Hasher.HashMessageWithMetadata(msg)
	var blockHash common.Hash
	copy(blockHash[:], hash[:])
	result := execution.MessageResult{
		BlockHash: blockHash,
	}

	m.Results[msgIdx] = &result
	if m.Latest < msgIdx {
		m.Latest = msgIdx
	}

	return containers.NewReadyPromise(&result, nil)
}

// HeadMessageIndex implements execution.ExecutionSequencer.
func (m *MockExecutionEngineForTransactionStreamer) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return containers.NewReadyPromise(m.Latest, nil)
}

// MarkFeedStart implements execution.ExecutionSequencer.
func (m *MockExecutionEngineForTransactionStreamer) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

// ErrorExecutionClientUnimplementedMethod is an error type that indicates that
// no message result was found for the given message index.
type ErrorNoMessageResultForIndex struct {
	MsgIdx arbutil.MessageIndex
}

// Error implements error
func (e ErrorNoMessageResultForIndex) Error() string {
	return fmt.Sprintf("no message result found for index %d", e.MsgIdx)
}

// ResultAtMessageIndex implements execution.ExecutionSequencer.
func (m *MockExecutionEngineForTransactionStreamer) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	result, resultOk := m.Results[msgIdx]
	if !resultOk {
		return containers.NewReadyPromise[*execution.MessageResult](nil, ErrorNoMessageResultForIndex{msgIdx})
	}

	return containers.NewReadyPromise(result, nil)
}
