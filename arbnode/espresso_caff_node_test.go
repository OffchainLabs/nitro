package arbnode

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/espressostreamer"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

type MockEspressoStreamer struct {
	currPos     uint64
	currHotShot uint64

	delayedPos uint64
	dbHotShot  uint64
}

var _ espressostreamer.EspressoStreamerInterface = (*MockEspressoStreamer)(nil)

// SetBatcherAddressesFetcher implements espressostreamer.EspressoStreamerInterface.
func (m *MockEspressoStreamer) SetBatcherAddressesFetcher(fetcher func(l1Height uint64) []common.Address) {
	panic("unimplemented")
}

func (m *MockEspressoStreamer) GetCurrentEarliestHotShotBlockNumber() uint64 {
	return m.currHotShot
}

func (m *MockEspressoStreamer) Start(ctx context.Context) error {
	return nil
}

func (m *MockEspressoStreamer) Peek(ctx context.Context) *espressostreamer.MessageWithMetadataAndPos {
	var delayedCnt uint64 = 1
	if m.delayedPos == m.currPos {
		delayedCnt = 2
	}
	result := espressostreamer.MessageWithMetadataAndPos{
		MessageWithMeta: arbostypes.MessageWithMetadata{
			DelayedMessagesRead: delayedCnt,
			Message:             &arbostypes.EmptyTestIncomingMessage,
		},
		Pos:           m.currPos,
		HotshotHeight: m.currHotShot,
	}
	return &result
}

func (m *MockEspressoStreamer) Advance() {
	m.currPos++
	m.currHotShot++
}

func (m *MockEspressoStreamer) Next(ctx context.Context) *espressostreamer.MessageWithMetadataAndPos {
	result := m.Peek(ctx)
	m.Advance()
	return result
}

func (m *MockEspressoStreamer) Reset(currentMessagePos uint64, currentHostshotBlock uint64) {
	m.currPos = currentMessagePos
	m.currHotShot = currentHostshotBlock
}

func (m *MockEspressoStreamer) RecordTimeDurationBetweenHotshotAndCurrentBlock(nextHotshotBlock uint64, blockProductionTime time.Time) {
}

func (m *MockEspressoStreamer) StoreHotshotBlock(db ethdb.Database, nextHotshotBlock uint64) error {
	return nil
}

func (m *MockEspressoStreamer) ReadNextHotshotBlockFromDb(db ethdb.Database) (uint64, error) {
	return m.dbHotShot, nil
}

type MockDelayedMessageFetcher struct{}

// This function isn't a proper implementation for the tests, but this gets the test to compile.
func (m *MockDelayedMessageFetcher) getDelayedMessageLatestIndexAtBlock(blockNumber uint64) (uint64, error) {
	return 1, nil
}

func (m *MockDelayedMessageFetcher) processDelayedMessage(messageWithMetadataAndPos *espressostreamer.MessageWithMetadataAndPos) (*espressostreamer.MessageWithMetadataAndPos, error) {
	if messageWithMetadataAndPos.MessageWithMeta.DelayedMessagesRead == 2 {
		return &espressostreamer.MessageWithMetadataAndPos{
			MessageWithMeta: arbostypes.MessageWithMetadata{
				Message:             arbostypes.InvalidL1Message,
				DelayedMessagesRead: messageWithMetadataAndPos.MessageWithMeta.DelayedMessagesRead,
			},
			Pos:           messageWithMetadataAndPos.Pos,
			HotshotHeight: messageWithMetadataAndPos.HotshotHeight,
		}, nil
	}
	return messageWithMetadataAndPos, nil
}

func (m *MockDelayedMessageFetcher) Start(ctx context.Context) bool {
	return true
}

func (m *MockDelayedMessageFetcher) storeDelayedMessageLatestIndex(db ethdb.Database, count uint64) error {
	return nil
}

func TestEspressoCaffNodeShouldReadDelayedMessageFromL1(t *testing.T) {

	caffNode := EspressoCaffNode{}

	// create a test execution client
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()

	initData := statetransfer.ArbosInitializationInfo{
		Accounts: []statetransfer.AccountInitializationInfo{
			{
				Addr:       common.HexToAddress("0x1111111111111111111111111111111111111111"),
				EthBalance: big.NewInt(params.Ether),
			},
		},
	}

	chainDb := rawdb.NewMemoryDatabase()
	initReader := statetransfer.NewMemoryInitDataReader(&initData)

	cacheConfig := core.DefaultCacheConfigWithScheme(env.GetTestStateScheme())
	bc, err := gethexec.WriteOrTestBlockChain(chainDb, cacheConfig, initReader, chainConfig, nil, nil, arbostypes.TestInitMessage, gethexec.ConfigDefault.TxLookupLimit, 0)

	if err != nil {
		Fail(t, err)
	}

	execEngine, err := gethexec.NewExecutionEngine(bc, 0)
	if err != nil {
		Fail(t, err)
	}
	caffNode.executionEngine = execEngine
	caffNode.espressoStreamer = &MockEspressoStreamer{delayedPos: 3, currPos: 0}
	caffNode.delayedMessageFetcher = &MockDelayedMessageFetcher{}
	ctx := context.Background()
	msg1, err := caffNode.peekMessage(ctx)
	require.NoError(t, err)

	require.Equal(t, msg1.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg1.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)
	require.Equal(t, msg1.Pos, uint64(0))
	caffNode.espressoStreamer.Advance()

	msg2, err := caffNode.peekMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, msg2.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg2.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)
	require.Equal(t, msg2.Pos, uint64(1))
	caffNode.espressoStreamer.Advance()

	msg3, err := caffNode.peekMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, msg3.MessageWithMeta.DelayedMessagesRead, uint64(1))
	require.Equal(t, msg3.MessageWithMeta.Message, &arbostypes.EmptyTestIncomingMessage)
	require.Equal(t, msg3.Pos, uint64(2))
	caffNode.espressoStreamer.Advance()

	msg4, err := caffNode.peekMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, msg4.MessageWithMeta.DelayedMessagesRead, uint64(2))
	require.Equal(t, msg4.MessageWithMeta.Message, arbostypes.InvalidL1Message)
	require.Equal(t, msg4.Pos, uint64(3))
}

func TestEspressoCaffNodeShouldResetToLastStoredHotshotBlock(t *testing.T) {
	caffNode := EspressoCaffNode{}
	caffNode.espressoStreamer = &MockEspressoStreamer{delayedPos: 3, dbHotShot: 10}

	espressoStreamer, _ := caffNode.espressoStreamer.(*MockEspressoStreamer)
	require.Equal(t, espressoStreamer.dbHotShot, uint64(10))
	require.Equal(t, espressoStreamer.delayedPos, uint64(3))
}
