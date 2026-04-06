// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"testing"

	"github.com/stretchr/testify/require"

	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func TestBatchDataSource_ErrorWhenNeitherSet(t *testing.T) {
	t.Parallel()
	n := &Node{}
	_, err := n.BatchDataSource()
	require.ErrorIs(t, err, ErrNoBatchDataReader)
}

func TestBatchDataSource_ReturnsInboxTracker(t *testing.T) {
	t.Parallel()
	tracker := &InboxTracker{}
	n := &Node{InboxTracker: tracker}
	got, err := n.BatchDataSource()
	require.NoError(t, err)
	require.Same(t, tracker, got)
}

func TestBatchDataSource_PrefersMessageExtractor(t *testing.T) {
	t.Parallel()
	extractor := &melrunner.MessageExtractor{}
	tracker := &InboxTracker{}
	n := &Node{MessageExtractor: extractor, InboxTracker: tracker}
	got, err := n.BatchDataSource()
	require.NoError(t, err)
	require.Same(t, extractor, got)
}

func TestGetL1Confirmations_NilReaderReturnsError(t *testing.T) {
	t.Parallel()
	// L1Reader must be non-nil to reach the BatchDataSource check.
	n := &Node{L1Reader: &headerreader.HeaderReader{}}
	p := n.GetL1Confirmations(0)
	_, err := p.Await(t.Context())
	require.ErrorIs(t, err, ErrNoBatchDataReader)
}

func TestGetL1Confirmations_NilL1ReaderReturnsNoError(t *testing.T) {
	t.Parallel()
	n := &Node{}
	p := n.GetL1Confirmations(0)
	val, err := p.Await(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(0), val)
}

func TestFindBatchContainingMessage_NilReaderReturnsError(t *testing.T) {
	t.Parallel()
	n := &Node{}
	p := n.FindBatchContainingMessage(0)
	_, err := p.Await(t.Context())
	require.ErrorIs(t, err, ErrNoBatchDataReader)
}
