package arbnode

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/espressostreamer"
)

type MockEspressoStreamer struct {
	currrPos    uint64
	currHotShot uint64

	delayedPos uint64
}

func (m *MockEspressoStreamer) Start(ctx context.Context) error {
	return nil
}

func (m *MockEspressoStreamer) Next() (*espressostreamer.MessageWithMetadataAndPos, error) {
	var delayedCnt uint64 = 1
	if m.delayedPos == m.currrPos {
		delayedCnt = 2
	}
	result := espressostreamer.MessageWithMetadataAndPos{
		MessageWithMeta: arbostypes.MessageWithMetadata{
			DelayedMessagesRead: delayedCnt,
			Message:             &arbostypes.EmptyTestIncomingMessage,
		},
		Pos:           m.currrPos,
		HotshotHeight: m.currHotShot,
	}
	m.currrPos++
	m.currHotShot++
	return &result, nil
}

func (m *MockEspressoStreamer) Reset(currentMessagePos uint64, currentHostshotBlock uint64) {}

func (m *MockEspressoStreamer) RecordTimeDurationBetweenHotshotAndCurrentBlock(nextHotshotBlock uint64, blockProductionTime time.Time) {
}

func (m *MockEspressoStreamer) StoreHotshotBlock(db ethdb.Database, nextHotshotBlock uint64) error {
	return nil
}

func (m *MockEspressoStreamer) ReadNextHotshotBlockFromDb(db ethdb.Database) (uint64, error) {
	return 0, nil
}

type MockDelayedMessageFetcher struct{}

// This function isn't a proper implementation for the tests, but this gets the test to compile.
func (m *MockDelayedMessageFetcher) getDelayedMessageCountAtBlock(blockNumber uint64) (uint64, error) {
	return 1, nil
}

func (m *MockDelayedMessageFetcher) processDelayedMessage(messageWithMetadataAndPos *espressostreamer.MessageWithMetadataAndPos) (*espressostreamer.MessageWithMetadataAndPos, error) {
	return messageWithMetadataAndPos, nil
}

func (m *MockDelayedMessageFetcher) reset(parentChainBlockNum uint64, seqNum uint64) {}

func TestEspressoCaffNodeShouldReadDelayedMessageFromL1(t *testing.T) {

	caffNode := EspressoCaffNode{}
	caffNode.espressoStreamer = &MockEspressoStreamer{delayedPos: 3}
	caffNode.delayedMessageFetcher = &MockDelayedMessageFetcher{}
	msg1, err := caffNode.nextMessage()
	require.NoError(t, err)

	require.Equal(t, msg1.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg1.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)

	msg2, err := caffNode.nextMessage()
	require.NoError(t, err)
	require.Equal(t, msg2.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg2.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)

	msg3, err := caffNode.nextMessage()
	require.NoError(t, err)
	require.Equal(t, msg3.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg3.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)

	msg4, err := caffNode.nextMessage()
	require.NoError(t, err)
	require.Equal(t, msg4.MessageWithMeta.DelayedMessagesRead, uint64(2))
	require.Equal(t, msg4.MessageWithMeta.Message, arbostypes.InvalidL1Message)
}
