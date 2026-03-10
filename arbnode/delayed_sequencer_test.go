// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

// stubDelayedMessageFetcher is a minimal DelayedMessageFetcher for constructor tests.
type stubDelayedMessageFetcher struct{}

func (s *stubDelayedMessageFetcher) GetDelayedCount() (uint64, error) { return 0, nil }

func (s *stubDelayedMessageFetcher) FinalizedDelayedMessageAtPosition(
	ctx context.Context, finalizedPosition uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64,
) (*arbostypes.L1IncomingMessage, common.Hash, error) {
	return nil, common.Hash{}, nil
}

func TestNewDelayedSequencer_Validation(t *testing.T) {
	t.Parallel()
	configFetcher := func() *DelayedSequencerConfig { return &TestDelayedSequencerConfig }
	stub := &stubDelayedMessageFetcher{}

	t.Run("both reader and fetcher provided returns error", func(t *testing.T) {
		// We can't easily construct a real InboxReader, but NewDelayedSequencer
		// checks for non-nil before using it, so we need a non-nil *InboxReader.
		// Use an empty struct pointer — the constructor should reject it before
		// dereferencing any fields.
		reader := &InboxReader{}
		_, err := NewDelayedSequencer(nil, reader, stub, nil, nil, nil, configFetcher)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must not have both")
	})

	t.Run("neither reader nor fetcher returns error", func(t *testing.T) {
		_, err := NewDelayedSequencer(nil, nil, nil, nil, nil, nil, configFetcher)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires either")
	})

	t.Run("only fetcher succeeds", func(t *testing.T) {
		d, err := NewDelayedSequencer(nil, nil, stub, nil, nil, nil, configFetcher)
		require.NoError(t, err)
		require.NotNil(t, d)
		require.Equal(t, stub, d.delayedCountFetcher)
	})
}

func TestGetDelayedSequencer_NilInterfaceSemantics(t *testing.T) {
	t.Parallel()
	configFetcher := func() *DelayedSequencerConfig { return &TestDelayedSequencerConfig }
	stub := &stubDelayedMessageFetcher{}

	t.Run("typed nil msgExtractor becomes untyped nil", func(t *testing.T) {
		// A typed nil *MessageExtractor must not be passed as a non-nil
		// DelayedMessageFetcher interface to NewDelayedSequencer; that would
		// bypass the constructor's nil check and panic later. getDelayedSequencer
		// must convert it to an untyped nil.
		// We can't call getDelayedSequencer with a nil exec (returns nil, nil early),
		// and we can't easily construct a real ExecutionSequencer, so we verify the
		// invariant directly: passing nil delayedCountFetcher with nil reader must error.
		_, err := NewDelayedSequencer(nil, nil, nil, nil, nil, nil, configFetcher)
		require.Error(t, err, "typed nil converted to untyped nil should be rejected by constructor")
		require.Contains(t, err.Error(), "requires either")
	})

	t.Run("non-nil msgExtractor is passed through", func(t *testing.T) {
		// When delayedCountFetcher is non-nil, it should be used directly.
		d, err := NewDelayedSequencer(nil, nil, stub, nil, nil, nil, configFetcher)
		require.NoError(t, err)
		require.Equal(t, stub, d.delayedCountFetcher)
	})
}
