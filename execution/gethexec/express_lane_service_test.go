// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/timeboost"
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
	tests := []struct {
		name        string
		es          *expressLaneService
		sub         *timeboost.ExpressLaneSubmission
		expectedErr error
		control     expressLaneControl
		valid       bool
	}{
		{
			name: "nil msg",
			sub:  nil,
			es: &expressLaneService{
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil tx",
			sub:  &timeboost.ExpressLaneSubmission{},
			es: &expressLaneService{
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil sig",
			sub: &timeboost.ExpressLaneSubmission{
				Transaction: &types.Transaction{},
			},
			es: &expressLaneService{
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "wrong chain id",
			es: &expressLaneService{
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			sub: &timeboost.ExpressLaneSubmission{
				ChainId:     big.NewInt(2),
				Transaction: &types.Transaction{},
				Signature:   []byte{'a'},
			},
			expectedErr: timeboost.ErrWrongChainId,
		},
		{
			name: "wrong auction contract",
			es: &expressLaneService{
				auctionContractAddr: common.Address{'a'},
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			sub: &timeboost.ExpressLaneSubmission{
				ChainId:                big.NewInt(1),
				AuctionContractAddress: common.Address{'b'},
				Transaction:            &types.Transaction{},
				Signature:              []byte{'b'},
			},
			expectedErr: timeboost.ErrWrongAuctionContract,
		},
		{
			name: "no onchain controller",
			es: &expressLaneService{
				auctionContractAddr: common.Address{'a'},
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			sub: &timeboost.ExpressLaneSubmission{
				ChainId:                big.NewInt(1),
				AuctionContractAddress: common.Address{'a'},
				Transaction:            &types.Transaction{},
				Signature:              []byte{'b'},
			},
			expectedErr: timeboost.ErrNoOnchainController,
		},
		{
			name: "bad round number",
			es: &expressLaneService{
				auctionContractAddr: common.Address{'a'},
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: common.Address{'b'},
			},
			sub: &timeboost.ExpressLaneSubmission{
				ChainId:                big.NewInt(1),
				AuctionContractAddress: common.Address{'a'},
				Transaction:            &types.Transaction{},
				Signature:              []byte{'b'},
				Round:                  100,
			},
			expectedErr: timeboost.ErrBadRoundNumber,
		},
		{
			name: "malformed signature",
			es: &expressLaneService{
				auctionContractAddr: common.Address{'a'},
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: common.Address{'b'},
			},
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
			name: "wrong signature",
			es: &expressLaneService{
				auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: common.Address{'b'},
			},
			sub:         buildInvalidSignatureSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6")),
			expectedErr: timeboost.ErrNotExpressLaneController,
		},
		{
			name: "not express lane controller",
			es: &expressLaneService{
				auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: common.Address{'b'},
			},
			sub:         buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv, 0),
			expectedErr: timeboost.ErrNotExpressLaneController,
		},
		{
			name: "OK",
			es: &expressLaneService{
				auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
				roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewCache[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: crypto.PubkeyToAddress(testPriv.PublicKey),
			},
			sub:   buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv, 0),
			valid: true,
		},
	}

	for _, _tt := range tests {
		tt := _tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.sub != nil && !errors.Is(tt.expectedErr, timeboost.ErrNoOnchainController) {
				tt.es.roundControl.Add(tt.sub.Round, &tt.control)
			}
			err := tt.es.validateExpressLaneTx(tt.sub)
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
	es := &expressLaneService{
		auctionContractAddr: auctionContractAddr,
		roundTimingInfo: timeboost.RoundTimingInfo{
			Offset:         time.Now(),
			Round:          time.Second * 10,
			AuctionClosing: time.Second * 5,
		},
		earlySubmissionGrace: time.Second * 2,
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
		roundControl: lru.NewCache[uint64, *expressLaneControl](8),
	}
	es.roundControl.Add(0, &expressLaneControl{
		controller: crypto.PubkeyToAddress(testPriv.PublicKey),
	})
	es.roundControl.Add(1, &expressLaneControl{
		controller: crypto.PubkeyToAddress(testPriv2.PublicKey),
	})

	sub1 := buildValidSubmission(t, auctionContractAddr, testPriv, 0)
	err := es.validateExpressLaneTx(sub1)
	require.NoError(t, err)

	// Send req for next round
	sub2 := buildValidSubmission(t, auctionContractAddr, testPriv2, 1)
	err = es.validateExpressLaneTx(sub2)
	require.ErrorIs(t, err, timeboost.ErrBadRoundNumber)

	// Sleep til 2 seconds before grace
	time.Sleep(time.Second * 6)
	err = es.validateExpressLaneTx(sub2)
	require.ErrorIs(t, err, timeboost.ErrBadRoundNumber)

	// Send req for next round within grace period
	time.Sleep(time.Second * 2)
	err = es.validateExpressLaneTx(sub2)
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

func (s *stubPublisher) PublishTimeboostedTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, txIsQueuedNotifier chan struct{}) error {
	defer close(txIsQueuedNotifier)
	if tx.Hash() != emptyTx.Hash() {
		return errors.New("oops, bad tx")
	}
	control, _ := s.els.roundControl.Get(0)
	s.publishedTxOrder = append(s.publishedTxOrder, control.sequence)
	return nil

}

func (s *stubPublisher) Config() *SequencerConfig {
	return &SequencerConfig{}
}

func Test_expressLaneService_sequenceExpressLaneSubmission_nonceTooLow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		msgAndResultBySequenceNumber: make(map[uint64]*msgAndResult),
		roundControl:                 lru.NewCache[uint64, *expressLaneControl](8),
	}
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	msg := &timeboost.ExpressLaneSubmission{
		SequenceNumber: 0,
	}

	err := els.sequenceExpressLaneSubmission(ctx, msg)
	require.ErrorIs(t, err, timeboost.ErrSequenceNumberTooLow)
}

func Test_expressLaneService_sequenceExpressLaneSubmission_duplicateNonce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:                 lru.NewCache[uint64, *expressLaneControl](8),
		msgAndResultBySequenceNumber: make(map[uint64]*msgAndResult),
		roundTimingInfo:              defaultTestRoundTimingInfo(time.Now()),
	}
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	msg := &timeboost.ExpressLaneSubmission{
		SequenceNumber: 2,
		Transaction:    types.NewTx(&types.DynamicFeeTx{Data: []byte{1}}),
	}
	var wg sync.WaitGroup
	wg.Add(3) // We expect only of the below two to return with an error here
	var err1, err2 error
	go func(w *sync.WaitGroup) {
		w.Done()
		err1 = els.sequenceExpressLaneSubmission(ctx, msg)
		wg.Done()
	}(&wg)
	go func(w *sync.WaitGroup) {
		w.Done()
		err2 = els.sequenceExpressLaneSubmission(ctx, msg)
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
	els := &expressLaneService{
		roundControl:                 lru.NewCache[uint64, *expressLaneControl](8),
		msgAndResultBySequenceNumber: make(map[uint64]*msgAndResult),
		roundTimingInfo:              defaultTestRoundTimingInfo(time.Now()),
	}
	els.StopWaiter.Start(ctx, els)
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})

	messages := []*timeboost.ExpressLaneSubmission{
		{
			SequenceNumber: 10,
			Transaction:    types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), []byte{1}),
		},
		{
			SequenceNumber: 5,
			Transaction:    emptyTx,
		},
		{
			SequenceNumber: 1,
			Transaction:    emptyTx,
		},
		{
			SequenceNumber: 4,
			Transaction:    emptyTx,
		},
		{
			SequenceNumber: 2,
			Transaction:    emptyTx,
		},
	}
	// We launch 5 goroutines out of which 2 would return with a result hence we initially add a delta of 7
	var wg sync.WaitGroup
	wg.Add(7)
	for _, msg := range messages {
		go func(w *sync.WaitGroup) {
			w.Done()
			err := els.sequenceExpressLaneSubmission(ctx, msg)
			if msg.SequenceNumber != 10 { // Because this go-routine will be interrupted after the test itself ends and 10 will still be waiting for result
				require.NoError(t, err)
				w.Done()
			}
		}(&wg)
	}
	wg.Wait()

	// We should have only published 2, as we are missing sequence number 3.
	time.Sleep(2 * time.Second)
	require.Equal(t, 2, len(stubPublisher.publishedTxOrder))
	els.Lock()
	require.Equal(t, 3, len(els.msgAndResultBySequenceNumber)) // Processed txs are deleted
	els.Unlock()

	wg.Add(2) // 4 & 5 should be able to get in after 3 so we add a delta of 2
	err := els.sequenceExpressLaneSubmission(ctx, &timeboost.ExpressLaneSubmission{SequenceNumber: 3, Transaction: emptyTx})
	require.NoError(t, err)
	wg.Wait()
	require.Equal(t, 5, len(stubPublisher.publishedTxOrder))

	els.Lock()
	require.Equal(t, 1, len(els.msgAndResultBySequenceNumber)) // Tx with seq num 10 should still be present
	els.Unlock()
}

func Test_expressLaneService_sequenceExpressLaneSubmission_erroredTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:                 lru.NewCache[uint64, *expressLaneControl](8),
		msgAndResultBySequenceNumber: make(map[uint64]*msgAndResult),
		roundTimingInfo:              defaultTestRoundTimingInfo(time.Now()),
	}
	els.StopWaiter.Start(ctx, els)
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	messages := []*timeboost.ExpressLaneSubmission{
		{
			SequenceNumber: 1,
			Transaction:    emptyTx,
		},
		{
			SequenceNumber: 2,
			Transaction:    types.NewTransaction(0, common.MaxAddress, big.NewInt(0), 0, big.NewInt(0), []byte{1}),
		},
		{
			SequenceNumber: 3,
			Transaction:    emptyTx,
		},
		{
			SequenceNumber: 4,
			Transaction:    emptyTx,
		},
	}
	for _, msg := range messages {
		if msg.Transaction.Hash() != emptyTx.Hash() {
			err := els.sequenceExpressLaneSubmission(ctx, msg)
			require.ErrorContains(t, err, "oops, bad tx")
		} else {
			err := els.sequenceExpressLaneSubmission(ctx, msg)
			require.NoError(t, err)
		}
	}
	// One tx out of the four should have failed, so we should have only published 3.
	// Since sequence number 2 failed after submission stage, that nonce is used up
	require.Equal(t, 3, len(stubPublisher.publishedTxOrder))
	require.Equal(t, []uint64{1, 3, 4}, stubPublisher.publishedTxOrder)
}

// TODO this test is just for RoundTimingInfo
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

func Benchmark_expressLaneService_validateExpressLaneTx(b *testing.B) {
	b.StopTimer()
	addr := crypto.PubkeyToAddress(testPriv.PublicKey)
	es := &expressLaneService{
		auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
		roundTimingInfo:     defaultTestRoundTimingInfo(time.Now()),
		roundControl:        lru.NewCache[uint64, *expressLaneControl](8),
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
	}
	es.roundControl.Add(0, &expressLaneControl{
		sequence:   1,
		controller: addr,
	})
	sub := buildValidSubmission(b, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv, 0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := es.validateExpressLaneTx(sub)
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
