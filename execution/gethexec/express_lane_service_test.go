// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var testPriv, testPriv2 *ecdsa.PrivateKey

func init() {
	privKey, err := crypto.HexToECDSA("93be75cc4df7acbb636b6abe6de2c0446235ac1dc7da9f290a70d83f088b486d")
	if err != nil {
		panic(err)
	}
	testPriv = privKey
	privKey2, err := crypto.HexToECDSA("93be75cc4df7acbb636b6abe6de2c0446235ac1dc7da9f290a70d83f088b486e")
	if err != nil {
		panic(err)
	}
	testPriv2 = privKey2
}

func defaultTestRoundTimingInfo(offset time.Time) timeboost.RoundTimingInfo {
	return timeboost.RoundTimingInfo{
		Offset:            offset,
		Round:             time.Minute,
		AuctionClosing:    time.Second * 15,
		ReserveSubmission: time.Second * 15,
	}
}

func Test_expressLaneService_validateExpressLaneTx(t *testing.T) {
	validSubmission := buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv, 0)
	tests := []struct {
		name        string
		t           *ExpressLaneTracker
		sub         *timeboost.ExpressLaneSubmission
		expectedErr error
		controller  common.Address
		valid       bool
	}{
		{
			name:        "nil msg",
			sub:         nil,
			t:           defaultTestTracker(),
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name:        "nil tx",
			sub:         &timeboost.ExpressLaneSubmission{},
			t:           defaultTestTracker(),
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil sig",
			sub: &timeboost.ExpressLaneSubmission{
				Transaction: &types.Transaction{},
			},
			t:           defaultTestTracker(),
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "oversized data",
			sub: func() *timeboost.ExpressLaneSubmission {
				submission := cloneSubmission(validSubmission)
				submission.Transaction = types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), make([]byte, DefaultSequencerConfig.MaxTxDataSize))
				return submission
			}(),
			t:           defaultTestTracker(),
			expectedErr: timeboost.ErrOversizedData,
		},
		{
			name: "wrong chain id",
			t:    defaultTestTrackerWithChainID(1),
			sub: func() *timeboost.ExpressLaneSubmission {
				submission := cloneSubmission(validSubmission)
				submission.ChainId = big.NewInt(2)
				return submission
			}(),
			expectedErr: timeboost.ErrWrongChainId,
		},
		{
			name: "wrong auction contract",
			t:    defaultTestTrackerWithConfig(common.Address{'a'}, defaultTestRoundTimingInfo(time.Now())),
			sub: func() *timeboost.ExpressLaneSubmission {
				submission := cloneSubmission(validSubmission)
				submission.AuctionContractAddress = common.Address{'b'}
				return submission
			}(),
			expectedErr: timeboost.ErrWrongAuctionContract,
		},
		{
			name:       "bad round number",
			t:          defaultTestTrackerWithConfig(common.Address{'a'}, defaultTestRoundTimingInfo(time.Now())),
			controller: common.Address{'b'},
			sub: func() *timeboost.ExpressLaneSubmission {
				submission := cloneSubmission(validSubmission)
				submission.AuctionContractAddress = common.Address{'a'}
				submission.Round = 100
				return submission
			}(),
			expectedErr: timeboost.ErrBadRoundNumber,
		},
		{
			name:       "malformed signature",
			t:          defaultTestTrackerWithConfig(common.Address{'a'}, defaultTestRoundTimingInfo(time.Now())),
			controller: common.Address{'b'},

			sub: &timeboost.ExpressLaneSubmission{
				ChainId:                big.NewInt(1),
				AuctionContractAddress: common.Address{'a'},
				Transaction:            types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
				Signature:              []byte{'b'},
				Round:                  0,
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name:        "wrong signature",
			t:           defaultTestTrackerWithConfig(common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), defaultTestRoundTimingInfo(time.Now())),
			controller:  common.Address{'b'},
			sub:         buildInvalidSignatureSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6")),
			expectedErr: timeboost.ErrNotExpressLaneController,
		},

		{
			name: "no onchain controller",
			t:    defaultTestTrackerWithConfig(common.Address{'a'}, defaultTestRoundTimingInfo(time.Now())),
			sub: func() *timeboost.ExpressLaneSubmission {
				submission := cloneSubmission(validSubmission)
				submission.AuctionContractAddress = common.Address{'a'}
				return submission
			}(),
			expectedErr: timeboost.ErrNoOnchainController,
		},
		{
			name:        "not express lane controller",
			t:           defaultTestTrackerWithConfig(common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), defaultTestRoundTimingInfo(time.Now())),
			controller:  common.Address{'b'},
			sub:         validSubmission,
			expectedErr: timeboost.ErrNotExpressLaneController,
		},
		{
			name:       "OK",
			t:          defaultTestTrackerWithConfig(common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), defaultTestRoundTimingInfo(time.Now())),
			controller: crypto.PubkeyToAddress(testPriv.PublicKey),
			sub:        validSubmission,
			valid:      true,
		},
	}

	for _, _tt := range tests {
		tt := _tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.sub != nil && !errors.Is(tt.expectedErr, timeboost.ErrNoOnchainController) {
				tt.t.roundControl.Store(tt.sub.Round, tt.controller)
			}
			err := tt.t.ValidateExpressLaneTx(tt.sub)
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func Test_expressLaneService_validateExpressLaneTx_gracePeriod(t *testing.T) {
	auctionContractAddr := common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6")
	tr := defaultTestTrackerWithConfig(common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), timeboost.RoundTimingInfo{
		Offset:         time.Now(),
		Round:          time.Second * 10,
		AuctionClosing: time.Second * 5,
	})
	tr.earlySubmissionGrace = time.Second * 2
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	tr.roundControl.Store(1, crypto.PubkeyToAddress(testPriv2.PublicKey))

	sub1 := buildValidSubmission(t, auctionContractAddr, testPriv, 0)
	err := tr.ValidateExpressLaneTx(sub1)
	require.NoError(t, err)

	// Send req for next round
	sub2 := buildValidSubmission(t, auctionContractAddr, testPriv2, 1)
	err = tr.ValidateExpressLaneTx(sub2)
	require.ErrorIs(t, err, timeboost.ErrBadRoundNumber)

	// Sleep til 2 seconds before grace
	time.Sleep(time.Second * 6)
	err = tr.ValidateExpressLaneTx(sub2)
	require.ErrorIs(t, err, timeboost.ErrBadRoundNumber)

	// Send req for next round within grace period
	time.Sleep(time.Second * 2)
	err = tr.ValidateExpressLaneTx(sub2)
	require.NoError(t, err)
}

type stubPublisher struct {
	els              *expressLaneService
	publishedTxOrder []uint64
}

func makeStubPublisher(els *expressLaneService) *stubPublisher {
	return &stubPublisher{
		els:              els,
		publishedTxOrder: make([]uint64, 0),
	}
}

var emptyTx = types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), nil)

type testTransactionPublisher struct {
	publishFunc func(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error
}

func (t testTransactionPublisher) PublishTimeboostedTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return t.publishFunc(parentCtx, tx, options)
}

func (s *stubPublisher) PublishTimeboostedTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	if tx.Hash() != emptyTx.Hash() {
		return errors.New("oops, bad tx")
	}
	s.publishedTxOrder = append(s.publishedTxOrder, 0)
	return nil
}

func Test_expressLaneService_sequenceExpressLaneSubmission_nonceTooLow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els := &expressLaneService{
		roundInfo: containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		tracker:   tr,
	}
	els.roundInfo.Add(0, &expressLaneRoundInfo{1, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	msg := buildValidSubmissionWithSeqAndTx(t, 0, 0, emptyTx)
	err := els.sequenceExpressLaneSubmission(msg)
	require.ErrorIs(t, err, timeboost.ErrSequenceNumberTooLow)
}

func Test_expressLaneService_sequenceExpressLaneSubmission_duplicateNonce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
	}
	var err error
	els.redisCoordinator, err = timeboost.NewRedisCoordinator(redisUrl, &timingInfo, 50)
	require.NoError(t, err)
	els.redisCoordinator.Start(ctx)
	els.roundInfo.Add(0, &expressLaneRoundInfo{1, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	msg1 := buildValidSubmissionWithSeqAndTx(t, 0, 2, types.NewTx(&types.DynamicFeeTx{Data: []byte{1}}))
	msg2 := buildValidSubmissionWithSeqAndTx(t, 0, 2, types.NewTx(&types.DynamicFeeTx{Data: []byte{2}}))
	var wg sync.WaitGroup
	wg.Add(2) // We expect only one of the two txs below to return with an error here
	var err1, err2 error
	go func(w *sync.WaitGroup) {
		err1 = els.sequenceExpressLaneSubmission(msg1)
		wg.Done()
	}(&wg)
	go func(w *sync.WaitGroup) {
		err2 = els.sequenceExpressLaneSubmission(msg2)
		wg.Done()
	}(&wg)
	wg.Wait()
	if err1 != nil && err2 != nil || err1 == nil && err2 == nil {
		t.Fatalf("cannot have err1 and err2 both nil or non-nil. err1: %v, err2: %v", err1, err2)
	}
	if err1 != nil {
		require.ErrorIs(t, err1, timeboost.ErrDuplicateSequenceNumber)
	} else {
		require.ErrorIs(t, err2, timeboost.ErrDuplicateSequenceNumber)
	}
	wg.Add(1) // As the goroutine that's still running will call wg.Done() after the test ends
}

func Test_expressLaneService_sequenceExpressLaneSubmission_outOfOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
	}
	var err error
	els.redisCoordinator, err = timeboost.NewRedisCoordinator(redisUrl, &timingInfo, 50)
	require.NoError(t, err)
	els.redisCoordinator.Start(ctx)
	els.roundInfo.Add(0, &expressLaneRoundInfo{1, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	messages := []*timeboost.ExpressLaneSubmission{
		buildValidSubmissionWithSeqAndTx(t, 0, 10, types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), []byte{1})),
		buildValidSubmissionWithSeqAndTx(t, 0, 5, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 1, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 4, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 2, emptyTx),
	}

	// We launch 5 goroutines and all would return with a result upon queuing or buffering
	var wg sync.WaitGroup
	wg.Add(5)
	for _, msg := range messages {
		go func(w *sync.WaitGroup) {
			err := els.sequenceExpressLaneSubmission(msg)
			require.NoError(t, err)
			w.Done()
		}(&wg)
	}
	wg.Wait()

	// We should have only published 1 and 2, as we are missing sequence number 3.
	require.Equal(t, 2, len(stubPublisher.publishedTxOrder))
	els.roundInfoMutex.Lock()
	roundInfo, _ := els.roundInfo.Get(0)
	require.Equal(t, uint64(3), roundInfo.sequence)
	require.Equal(t, 5, len(roundInfo.msgBySequenceNumber))
	els.roundInfoMutex.Unlock()

	// 4 & 5 should be able to get in after 3 so we add a delta of 2
	err = els.sequenceExpressLaneSubmission(buildValidSubmissionWithSeqAndTx(t, 0, 3, emptyTx))
	require.NoError(t, err)
	require.Equal(t, 5, len(stubPublisher.publishedTxOrder))
	els.roundInfoMutex.Lock()
	roundInfo, _ = els.roundInfo.Get(0)
	require.Equal(t, uint64(6), roundInfo.sequence)
	require.Equal(t, 6, len(roundInfo.msgBySequenceNumber))
	els.roundInfoMutex.Unlock()
}

func Test_expressLaneService_sequenceExpressLaneSubmission_erroredTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &SequencerConfig{} },
		tracker:         tr,
	}
	var err error
	els.redisCoordinator, err = timeboost.NewRedisCoordinator(redisUrl, &timingInfo, 50)
	require.NoError(t, err)
	els.redisCoordinator.Start(ctx)
	els.roundInfo.Add(0, &expressLaneRoundInfo{1, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	messages := []*timeboost.ExpressLaneSubmission{
		buildValidSubmissionWithSeqAndTx(t, 0, 1, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 2, types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), []byte{1})),
		buildValidSubmissionWithSeqAndTx(t, 0, 3, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 4, emptyTx),
	}
	for _, msg := range messages {
		if msg.Transaction.Hash() != emptyTx.Hash() {
			err := els.sequenceExpressLaneSubmission(msg)
			require.ErrorContains(t, err, "oops, bad tx")
		} else {
			err := els.sequenceExpressLaneSubmission(msg)
			require.NoError(t, err)
		}
	}

	// One tx out of the four should have failed, so we should have only published 3.
	// Since sequence number 2 failed after submission stage, that nonce is used up
	require.Equal(t, 3, len(stubPublisher.publishedTxOrder))
}

func Test_expressLaneService_syncFromRedis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	redisUrl := redisutil.CreateTestRedis(ctx, t)
	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els1 := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
	}
	var err error
	els1.redisCoordinator, err = timeboost.NewRedisCoordinator(redisUrl, &timingInfo, 50)
	require.NoError(t, err)
	els1.redisCoordinator.Start(ctx)

	els1.roundInfo.Add(0, &expressLaneRoundInfo{1, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els1.StopWaiter.Start(ctx, els1)
	stubPublisher1 := makeStubPublisher(els1)
	els1.transactionPublisher = stubPublisher1

	messages := []*timeboost.ExpressLaneSubmission{
		buildValidSubmissionWithSeqAndTx(t, 0, 1, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 3, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 4, emptyTx),
		buildValidSubmissionWithSeqAndTx(t, 0, 5, emptyTx),
	}

	// We launch 4 goroutines and all would return with a result upon queuing or buffering
	var wg sync.WaitGroup
	wg.Add(4)
	for _, msg := range messages {
		go func(w *sync.WaitGroup) {
			_ = els1.sequenceExpressLaneSubmission(msg)
			w.Done()
		}(&wg)
	}
	wg.Wait()

	// Only one tx out of the three should have been processed
	require.Equal(t, 1, len(stubPublisher1.publishedTxOrder))

	time.Sleep(time.Second) // wait for parallel redis update threads to complete

	tr2 := defaultTestTracker()
	tr2.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))
	els2 := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr2,
	}
	els2.redisCoordinator, err = timeboost.NewRedisCoordinator(redisUrl, &timingInfo, 50)
	require.NoError(t, err)
	els2.redisCoordinator.Start(ctx)

	els2.StopWaiter.Start(ctx, els1)
	stubPublisher2 := makeStubPublisher(els2)
	els2.transactionPublisher = stubPublisher2

	// As els2 becomes an active sequencer, syncFromRedis would be called when Activate() function of sequencer is invoked
	els2.syncFromRedis()

	els2.roundInfoMutex.Lock()
	roundInfo, exists := els2.roundInfo.Get(0)
	if !exists {
		t.Fatal("missing roundInfo")
	}
	if roundInfo.sequence != 2 {
		t.Fatalf("round sequence count mismatch. Want: 2, Got: %d", roundInfo.sequence)
	}
	if len(roundInfo.msgBySequenceNumber) != 3 { // There should be three pending txs in msgAndResult map
		t.Fatalf("number of future sequence txs mismatch. Want: 3, Got: %d", len(roundInfo.msgBySequenceNumber))
	}
	els2.roundInfoMutex.Unlock()

	err = els2.sequenceExpressLaneSubmission(buildValidSubmissionWithSeqAndTx(t, 0, 2, emptyTx)) // Send an unblocking tx
	require.NoError(t, err)

	time.Sleep(time.Second) // wait for future seq num txs to be processed

	// Check that all pending txs are sequenced
	require.Equal(t, 4, len(stubPublisher2.publishedTxOrder))

	// Check final state of roundInfo
	els2.roundInfoMutex.Lock()
	roundInfo, exists = els2.roundInfo.Get(0)
	if !exists {
		t.Fatal("missing roundInfo")
	}
	if roundInfo.sequence != 6 {
		t.Fatalf("round sequence count mismatch. Want: 6, Got: %d", roundInfo.sequence)
	}
	els2.roundInfoMutex.Unlock()
}

func TestIsWithinAuctionCloseWindow(t *testing.T) {
	initialTimestamp := time.Date(2024, 8, 8, 15, 0, 0, 0, time.UTC)
	roundTimingInfo := defaultTestRoundTimingInfo(initialTimestamp)

	tests := []struct {
		name         string
		arrivalTime  time.Time
		expectedBool bool
	}{
		{
			name:         "Right before auction close window",
			arrivalTime:  initialTimestamp.Add(44 * time.Second), // 16 seconds left to the next round
			expectedBool: false,
		},
		{
			name:         "On the edge of auction close window",
			arrivalTime:  initialTimestamp.Add(45 * time.Second), // Exactly 15 seconds left to the next round
			expectedBool: true,
		},
		{
			name:         "Outside auction close window",
			arrivalTime:  initialTimestamp.Add(30 * time.Second), // 30 seconds left to the next round
			expectedBool: false,
		},
		{
			name:         "Exactly at the next round",
			arrivalTime:  initialTimestamp.Add(time.Minute), // At the start of the next round
			expectedBool: false,
		},
		{
			name:         "Just before the start of the next round",
			arrivalTime:  initialTimestamp.Add(time.Minute - 1*time.Second), // 1 second left to the next round
			expectedBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := roundTimingInfo.IsWithinAuctionCloseWindow(tt.arrivalTime)
			if actual != tt.expectedBool {
				t.Errorf("IsWithinAuctionCloseWindow(%v) = %v; want %v", tt.arrivalTime, actual, tt.expectedBool)
			}
		})
	}
}

func Test_expressLaneService_dontCareSequence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))

	mp := &struct {
		processedDontCare bool
		processedTx       *types.Transaction
		processedOptions  *arbitrum_types.ConditionalOptions
	}{}

	mockPublish := func(ctx context.Context, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions) error {
		mp.processedTx = tx
		mp.processedDontCare = true
		return nil
	}

	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
		transactionPublisher: testTransactionPublisher{
			publishFunc: mockPublish,
		},
	}

	els.StopWaiter.Start(ctx, els)

	// Test with a transaction that uses DontCareSequence
	tx := types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), []byte("dontcare"))
	dontCareMsg := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, tx)

	err := els.sequenceExpressLaneSubmission(dontCareMsg)
	require.NoError(t, err)

	require.True(t, mp.processedDontCare)
	require.Equal(t, tx.Hash(), mp.processedTx.Hash())
}

// Test_expressLaneService_mixedSequenceNumbersDontCareFirst tests sending dontcare sequence numbers first, then normal sequence numbers
func Test_expressLaneService_mixedSequenceNumbersDontCareFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))

	publishedTxs := []*types.Transaction{}
	mockPublish := func(ctx context.Context, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions) error {
		publishedTxs = append(publishedTxs, tx)
		return nil
	}

	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
		transactionPublisher: testTransactionPublisher{
			publishFunc: mockPublish,
		},
	}

	els.roundInfo.Add(0, &expressLaneRoundInfo{0, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)

	// First send transactions with DontCareSequence numbers
	dontCareTx1 := types.NewTransaction(0, common.Address{1}, big.NewInt(1), 0, big.NewInt(0), []byte("dontcare1"))
	dontCareTx2 := types.NewTransaction(0, common.Address{2}, big.NewInt(2), 0, big.NewInt(0), []byte("dontcare2"))

	dontCareMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx1)
	dontCareMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx2)

	// Then send transactions with normal sequence numbers
	normalTx1 := types.NewTransaction(0, common.Address{3}, big.NewInt(3), 0, big.NewInt(0), []byte("normal1"))
	normalTx2 := types.NewTransaction(0, common.Address{4}, big.NewInt(4), 0, big.NewInt(0), []byte("normal2"))

	normalMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, 0, normalTx1)
	normalMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, 1, normalTx2)

	// Submit the messages
	err := els.sequenceExpressLaneSubmission(dontCareMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(dontCareMsg2)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg2)
	require.NoError(t, err)

	// All 4 transactions should be published
	require.Equal(t, 4, len(publishedTxs))

	// Check that the transactions were published in the expected order
	require.Equal(t, dontCareTx1.Hash(), publishedTxs[0].Hash())
	require.Equal(t, dontCareTx2.Hash(), publishedTxs[1].Hash())
	require.Equal(t, normalTx1.Hash(), publishedTxs[2].Hash())
	require.Equal(t, normalTx2.Hash(), publishedTxs[3].Hash())

	// Check that the sequence number was updated correctly
	els.roundInfoMutex.Lock()
	roundInfo, _ := els.roundInfo.Get(0)
	require.Equal(t, uint64(2), roundInfo.sequence) // Should be 2 after processing seq 0 and 1
	els.roundInfoMutex.Unlock()
}

// Test_expressLaneService_mixedSequenceNumbersNormalFirst tests sending normal sequence numbers first, then dontcare sequence numbers
func Test_expressLaneService_mixedSequenceNumbersNormalFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))

	publishedTxs := []*types.Transaction{}
	mockPublish := func(ctx context.Context, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions) error {
		publishedTxs = append(publishedTxs, tx)
		return nil
	}

	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
		transactionPublisher: testTransactionPublisher{
			publishFunc: mockPublish,
		},
	}

	els.roundInfo.Add(0, &expressLaneRoundInfo{0, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)

	// First send transactions with normal sequence numbers
	normalTx1 := types.NewTransaction(0, common.Address{1}, big.NewInt(1), 0, big.NewInt(0), []byte("normal1"))
	normalTx2 := types.NewTransaction(0, common.Address{2}, big.NewInt(2), 0, big.NewInt(0), []byte("normal2"))

	normalMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, 0, normalTx1)
	normalMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, 1, normalTx2)

	// Then send transactions with DontCareSequence numbers
	dontCareTx1 := types.NewTransaction(0, common.Address{3}, big.NewInt(3), 0, big.NewInt(0), []byte("dontcare1"))
	dontCareTx2 := types.NewTransaction(0, common.Address{4}, big.NewInt(4), 0, big.NewInt(0), []byte("dontcare2"))

	dontCareMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx1)
	dontCareMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx2)

	// Submit the messages
	err := els.sequenceExpressLaneSubmission(normalMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg2)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(dontCareMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(dontCareMsg2)
	require.NoError(t, err)

	// All 4 transactions should be published
	require.Equal(t, 4, len(publishedTxs))

	// Check that the transactions were published in the expected order
	require.Equal(t, normalTx1.Hash(), publishedTxs[0].Hash())
	require.Equal(t, normalTx2.Hash(), publishedTxs[1].Hash())
	require.Equal(t, dontCareTx1.Hash(), publishedTxs[2].Hash())
	require.Equal(t, dontCareTx2.Hash(), publishedTxs[3].Hash())

	// Check that the sequence number was updated correctly
	els.roundInfoMutex.Lock()
	roundInfo, _ := els.roundInfo.Get(0)
	require.Equal(t, uint64(2), roundInfo.sequence) // Should be 2 after processing seq 0 and 1
	els.roundInfoMutex.Unlock()
}

// Test_expressLaneService_mixedSequenceNumbersIntermixed tests sending a mix of normal and dontcare sequence numbers
func Test_expressLaneService_mixedSequenceNumbersIntermixed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))

	publishedTxs := []*types.Transaction{}
	mockPublish := func(ctx context.Context, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions) error {
		publishedTxs = append(publishedTxs, tx)
		return nil
	}

	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
		transactionPublisher: testTransactionPublisher{
			publishFunc: mockPublish,
		},
	}

	els.roundInfo.Add(0, &expressLaneRoundInfo{0, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)

	// Create transactions with mixed sequence numbers
	normalTx1 := types.NewTransaction(0, common.Address{1}, big.NewInt(1), 0, big.NewInt(0), []byte("normal1"))
	dontCareTx1 := types.NewTransaction(0, common.Address{2}, big.NewInt(2), 0, big.NewInt(0), []byte("dontcare1"))
	normalTx2 := types.NewTransaction(0, common.Address{3}, big.NewInt(3), 0, big.NewInt(0), []byte("normal2"))
	dontCareTx2 := types.NewTransaction(0, common.Address{4}, big.NewInt(4), 0, big.NewInt(0), []byte("dontcare2"))
	normalTx3 := types.NewTransaction(0, common.Address{5}, big.NewInt(5), 0, big.NewInt(0), []byte("normal3"))

	normalMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, 0, normalTx1)
	dontCareMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx1)
	normalMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, 1, normalTx2)
	dontCareMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx2)
	normalMsg3 := buildValidSubmissionWithSeqAndTx(t, 0, 2, normalTx3)

	// Submit the messages in an intermixed order
	err := els.sequenceExpressLaneSubmission(normalMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(dontCareMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg2)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(dontCareMsg2)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg3)
	require.NoError(t, err)

	// All 5 transactions should be published
	require.Equal(t, 5, len(publishedTxs))

	// Check that the transactions were published in the correct order
	require.Equal(t, normalTx1.Hash(), publishedTxs[0].Hash())
	require.Equal(t, dontCareTx1.Hash(), publishedTxs[1].Hash())
	require.Equal(t, normalTx2.Hash(), publishedTxs[2].Hash())
	require.Equal(t, dontCareTx2.Hash(), publishedTxs[3].Hash())
	require.Equal(t, normalTx3.Hash(), publishedTxs[4].Hash())

	// Check that the sequence number was updated correctly
	els.roundInfoMutex.Lock()
	roundInfo, _ := els.roundInfo.Get(0)
	require.Equal(t, uint64(3), roundInfo.sequence) // Should be 3 after processing seq 0, 1, and 2
	els.roundInfoMutex.Unlock()
}

// Test_expressLaneService_dontCareWithQueuedTransactions tests that dontcare transactions are processed immediately
// even when regular sequence numbers are queued
func Test_expressLaneService_dontCareWithQueuedTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timingInfo := defaultTestRoundTimingInfo(time.Now())
	tr := defaultTestTracker()
	tr.roundControl.Store(0, crypto.PubkeyToAddress(testPriv.PublicKey))

	publishedTxs := []*types.Transaction{}
	mockPublish := func(ctx context.Context, tx *types.Transaction, _ *arbitrum_types.ConditionalOptions) error {
		publishedTxs = append(publishedTxs, tx)
		return nil
	}

	els := &expressLaneService{
		roundInfo:       containers.NewLruCache[uint64, *expressLaneRoundInfo](8),
		roundTimingInfo: timingInfo,
		seqConfig:       func() *SequencerConfig { return &DefaultSequencerConfig },
		tracker:         tr,
		transactionPublisher: testTransactionPublisher{
			publishFunc: mockPublish,
		},
	}

	els.roundInfo.Add(0, &expressLaneRoundInfo{0, make(map[uint64]*timeboost.ExpressLaneSubmission)})
	els.StopWaiter.Start(ctx, els)

	// Create some transactions with gaps in sequence numbers
	normalTx1 := types.NewTransaction(0, common.Address{1}, big.NewInt(1), 0, big.NewInt(0), []byte("normal1"))
	normalTx3 := types.NewTransaction(0, common.Address{3}, big.NewInt(3), 0, big.NewInt(0), []byte("normal3"))
	normalTx4 := types.NewTransaction(0, common.Address{4}, big.NewInt(4), 0, big.NewInt(0), []byte("normal4"))
	dontCareTx1 := types.NewTransaction(0, common.Address{5}, big.NewInt(5), 0, big.NewInt(0), []byte("dontcare1"))
	normalTx2 := types.NewTransaction(0, common.Address{2}, big.NewInt(2), 0, big.NewInt(0), []byte("normal2"))

	normalMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, 0, normalTx1)
	normalMsg3 := buildValidSubmissionWithSeqAndTx(t, 0, 2, normalTx3)
	normalMsg4 := buildValidSubmissionWithSeqAndTx(t, 0, 3, normalTx4)
	dontCareMsg1 := buildValidSubmissionWithSeqAndTx(t, 0, DontCareSequence, dontCareTx1)
	normalMsg2 := buildValidSubmissionWithSeqAndTx(t, 0, 1, normalTx2)

	// Submit the transactions with a gap in sequence numbers
	err := els.sequenceExpressLaneSubmission(normalMsg1)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg3)
	require.NoError(t, err)

	err = els.sequenceExpressLaneSubmission(normalMsg4)
	require.NoError(t, err)

	// At this point, only normalTx1 should be published because of the gap at sequence number 1
	require.Equal(t, 1, len(publishedTxs))
	require.Equal(t, normalTx1.Hash(), publishedTxs[0].Hash())

	// Submit a dontcare transaction - it should be processed immediately
	err = els.sequenceExpressLaneSubmission(dontCareMsg1)
	require.NoError(t, err)

	// Now dontCareTx1 should also be published, but normalTx3 and normalTx4 should still be queued
	require.Equal(t, 2, len(publishedTxs))
	require.Equal(t, dontCareTx1.Hash(), publishedTxs[1].Hash())

	// Now fill the gap with normalMsg2
	err = els.sequenceExpressLaneSubmission(normalMsg2)
	require.NoError(t, err)

	// Now all transactions should be published
	require.Equal(t, 5, len(publishedTxs))
	require.Equal(t, normalTx1.Hash(), publishedTxs[0].Hash())
	require.Equal(t, dontCareTx1.Hash(), publishedTxs[1].Hash())
	require.Equal(t, normalTx2.Hash(), publishedTxs[2].Hash())
	require.Equal(t, normalTx3.Hash(), publishedTxs[3].Hash())
	require.Equal(t, normalTx4.Hash(), publishedTxs[4].Hash())

	// Check that the sequence number was updated correctly
	els.roundInfoMutex.Lock()
	roundInfo, _ := els.roundInfo.Get(0)
	require.Equal(t, uint64(4), roundInfo.sequence) // Should be 4 after processing seq 0, 1, 2, and 3
	els.roundInfoMutex.Unlock()
}

func Benchmark_expressLaneService_validateExpressLaneTx(b *testing.B) {
	b.StopTimer()
	addr := crypto.PubkeyToAddress(testPriv.PublicKey)
	tr := defaultTestTrackerWithConfig(common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), defaultTestRoundTimingInfo(time.Now()))
	tr.roundControl.Store(0, addr)

	sub := buildValidSubmission(b, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv, 0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := tr.ValidateExpressLaneTx(sub)
		require.NoError(b, err)
	}
}

func buildSignature(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	prefixedData := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data))), data...))
	signature, err := crypto.Sign(prefixedData, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func buildInvalidSignatureSubmission(
	t *testing.T,
	auctionContractAddr common.Address,
) *timeboost.ExpressLaneSubmission {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	b := &timeboost.ExpressLaneSubmission{
		ChainId:                big.NewInt(1),
		AuctionContractAddress: auctionContractAddr,
		Transaction:            types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
		Signature:              make([]byte, 65),
		Round:                  0,
	}
	other := &timeboost.ExpressLaneSubmission{
		ChainId:                big.NewInt(2),
		AuctionContractAddress: auctionContractAddr,
		Transaction:            types.NewTransaction(320, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
		Signature:              make([]byte, 65),
		Round:                  30,
	}
	otherData, err := other.ToMessageBytes()
	require.NoError(t, err)
	signature, err := buildSignature(privateKey, otherData)
	require.NoError(t, err)
	b.Signature = signature
	return b
}

func buildValidSubmission(
	t testing.TB,
	auctionContractAddr common.Address,
	privKey *ecdsa.PrivateKey,
	round uint64,
) *timeboost.ExpressLaneSubmission {
	b := &timeboost.ExpressLaneSubmission{
		ChainId:                big.NewInt(1),
		AuctionContractAddress: auctionContractAddr,
		Transaction:            types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
		Signature:              make([]byte, 65),
		Round:                  round,
	}
	data, err := b.ToMessageBytes()
	require.NoError(t, err)
	signature, err := buildSignature(privKey, data)
	require.NoError(t, err)
	b.Signature = signature
	return b
}

func buildValidSubmissionWithSeqAndTx(
	t testing.TB,
	round uint64,
	seq uint64,
	tx *types.Transaction,
) *timeboost.ExpressLaneSubmission {
	b := &timeboost.ExpressLaneSubmission{
		ChainId:                big.NewInt(1),
		AuctionContractAddress: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
		Transaction:            tx,
		Signature:              make([]byte, 65),
		Round:                  round,
		SequenceNumber:         seq,
	}
	data, err := b.ToMessageBytes()
	require.NoError(t, err)
	signature, err := buildSignature(testPriv, data)
	require.NoError(t, err)
	b.Signature = signature
	return b
}

func cloneSubmission(original *timeboost.ExpressLaneSubmission) *timeboost.ExpressLaneSubmission {
	return &timeboost.ExpressLaneSubmission{
		ChainId:                new(big.Int).Set(original.ChainId),
		AuctionContractAddress: original.AuctionContractAddress,
		Transaction:            original.Transaction,
		Signature:              append([]byte{}, original.Signature...),
		Round:                  original.Round,
		SequenceNumber:         original.SequenceNumber,
	}
}

func defaultTestTracker() *ExpressLaneTracker {
	return &ExpressLaneTracker{
		maxTxSize: uint64(DefaultSequencerConfig.MaxTxDataSize), // #nosec G115
	}
}

func defaultTestTrackerWithConfig(
	auctionAddr common.Address,
	roundTimingInfo timeboost.RoundTimingInfo,
) *ExpressLaneTracker {
	return &ExpressLaneTracker{
		auctionContractAddr: auctionAddr,
		roundTimingInfo:     roundTimingInfo,
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
		maxTxSize: uint64(DefaultSequencerConfig.MaxTxDataSize), // #nosec G115
	}
}

func defaultTestTrackerWithChainID(chainID int64) *ExpressLaneTracker {
	return &ExpressLaneTracker{
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(chainID),
		},
		maxTxSize: uint64(DefaultSequencerConfig.MaxTxDataSize), // #nosec G115
	}
}
