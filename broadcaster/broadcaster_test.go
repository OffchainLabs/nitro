// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcaster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

const chainId = uint64(5555)

type predicate interface {
	Test() bool
	Error() string
}

func waitUntilUpdated(t *testing.T, p predicate) {
	t.Helper()
	updateTimer := time.NewTimer(2 * time.Second)
	defer updateTimer.Stop()
	for {
		if p.Test() {
			break
		}
		select {
		case <-updateTimer.C:
			t.Fatalf("%s", p.Error())
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type messageCountPredicate struct {
	b              *Broadcaster
	expected       int
	contextMessage string
	was            int
}

func (p *messageCountPredicate) Test() bool {
	p.was = p.b.GetCachedMessageCount()
	return p.was == p.expected
}

func (p *messageCountPredicate) Error() string {
	return fmt.Sprintf("Expected %d, was %d: %s", p.expected, p.was, p.contextMessage)
}

func testMessage() arbostypes.MessageWithMetadataAndBlockInfo {
	return arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta:    arbostypes.EmptyTestMessageWithMetadata,
		BlockHash:          nil,
		BlockMetadata:      nil,
		ArbOSVersionBefore: 0,
	}
}

func TestBroadcasterMessagesRemovedOnConfirmation(t *testing.T) {
	b, cancelFunc, _ := setup(t)
	defer cancelFunc()
	defer b.StopAndWait()

	expectMessageCount := func(count int, contextMessage string) predicate {
		return &messageCountPredicate{b, count, contextMessage, 0}
	}

	// Normal broadcasting and confirming
	Require(t, b.BroadcastSingle(testMessage(), 1))
	waitUntilUpdated(t, expectMessageCount(1, "after 1 message"))
	Require(t, b.BroadcastSingle(testMessage(), 2))
	waitUntilUpdated(t, expectMessageCount(2, "after 2 messages"))
	Require(t, b.BroadcastSingle(testMessage(), 3))
	waitUntilUpdated(t, expectMessageCount(3, "after 3 messages"))
	Require(t, b.BroadcastSingle(testMessage(), 4))
	waitUntilUpdated(t, expectMessageCount(4, "after 4 messages"))
	Require(t, b.BroadcastSingle(testMessage(), 5))
	waitUntilUpdated(t, expectMessageCount(5, "after 4 messages"))
	Require(t, b.BroadcastSingle(testMessage(), 6))
	waitUntilUpdated(t, expectMessageCount(6, "after 4 messages"))

	b.Confirm(4)
	waitUntilUpdated(t, expectMessageCount(2,
		"after 6 messages, 4 cleared by confirm"))

	b.Confirm(5)
	waitUntilUpdated(t, expectMessageCount(1,
		"after 6 messages, 5 cleared by confirm"))

	b.Confirm(4)
	waitUntilUpdated(t, expectMessageCount(1,
		"nothing changed because confirmed sequence number before cache"))

	b.Confirm(5)
	Require(t, b.BroadcastSingle(testMessage(), 7))
	waitUntilUpdated(t, expectMessageCount(2,
		"after 7 messages, 5 cleared by confirm"))

	// Confirm not-yet-seen or already confirmed/cleared sequence numbers twice to force clearing cache
	b.Confirm(8)
	waitUntilUpdated(t, expectMessageCount(0,
		"clear all messages after confirmed 1 beyond latest"))
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func TestBatchDataStatsIsIncludedBasedOnArbOSVersion(t *testing.T) {
	b, cancelFunc, signer := setup(t)
	defer cancelFunc()
	defer b.StopAndWait()

	sequenceNumber := arbutil.MessageIndex(0)
	message := testMessage()
	batchDataStats := &arbostypes.BatchDataStats{Length: 1, NonZeros: 2}
	message.MessageWithMeta.Message.BatchDataStats = batchDataStats

	// For ArbOS versions >= 50, BatchDataStats should be preserved
	message.ArbOSVersionBefore = params.ArbosVersion_50
	feedMsg, err := b.NewBroadcastFeedMessage(message, sequenceNumber)
	Require(t, err)
	require.Equal(t, batchDataStats, feedMsg.Message.Message.BatchDataStats)
	require.Equal(t, signMessage(t, message, sequenceNumber, signer), feedMsg.Signature)

	// For ArbOS versions < 50, BatchDataStats should be nil
	message.ArbOSVersionBefore = params.ArbosVersion_41
	feedMsg, err = b.NewBroadcastFeedMessage(message, sequenceNumber)
	Require(t, err)
	require.Nil(t, feedMsg.Message.Message.BatchDataStats)

	message.MessageWithMeta.Message.BatchDataStats = nil
	require.Equal(t, signMessage(t, message, sequenceNumber, signer), feedMsg.Signature)
}

func setup(t *testing.T) (*Broadcaster, context.CancelFunc, signature.DataSignerFunc) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	config := wsbroadcastserver.DefaultTestBroadcasterConfig

	feedErrChan := make(chan error, 10)
	signer := dataSigner(t)
	b := NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config }, chainId, feedErrChan, signer)
	Require(t, b.Initialize())
	Require(t, b.Start(ctx))

	return b, cancelFunc, signer
}

func dataSigner(t *testing.T) signature.DataSignerFunc {
	testPrivateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)
	return signature.DataSignerFromPrivateKey(testPrivateKey)
}

func signMessage(t *testing.T, message arbostypes.MessageWithMetadataAndBlockInfo, sequenceNumber arbutil.MessageIndex, signer signature.DataSignerFunc) []byte {
	hash, err := message.MessageWithMeta.Hash(sequenceNumber, chainId)
	Require(t, err)
	sig, err := signer(hash.Bytes())
	Require(t, err)
	return sig
}
