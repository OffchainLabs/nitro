// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
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
				initialTimestamp:    time.Now(),
				roundDuration:       time.Minute,
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
				initialTimestamp:    time.Now(),
				roundDuration:       time.Minute,
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
				initialTimestamp:    time.Now(),
				roundDuration:       time.Minute,
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
				initialTimestamp:    time.Now(),
				roundDuration:       time.Minute,
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
				initialTimestamp:    time.Now(),
				roundDuration:       time.Minute,
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
			if tt.sub != nil {
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
		auctionContractAddr:  auctionContractAddr,
		initialTimestamp:     time.Now(),
		roundDuration:        time.Second * 10,
		auctionClosing:       time.Second * 5,
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

func (s *stubPublisher) PublishTimeboostedTransaction(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	if tx == nil {
		return errors.New("oops, bad tx")
	}
	control, _ := s.els.roundControl.Get(0)
	s.publishedTxOrder = append(s.publishedTxOrder, control.sequence)
	return nil

}

func Test_expressLaneService_sequenceExpressLaneSubmission_nonceTooLow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
		roundControl:             lru.NewCache[uint64, *expressLaneControl](8),
	}
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
		roundControl:             lru.NewCache[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	msg := &timeboost.ExpressLaneSubmission{
		SequenceNumber: 2,
	}
	err := els.sequenceExpressLaneSubmission(ctx, msg)
	require.NoError(t, err)
	// Because the message is for a future sequence number, it
	// should get queued, but not yet published.
	require.Equal(t, 0, len(stubPublisher.publishedTxOrder))
	// Sending it again should give us an error.
	err = els.sequenceExpressLaneSubmission(ctx, msg)
	require.ErrorIs(t, err, timeboost.ErrDuplicateSequenceNumber)
}

func Test_expressLaneService_sequenceExpressLaneSubmission_outOfOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:             lru.NewCache[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})

	messages := []*timeboost.ExpressLaneSubmission{
		{
			SequenceNumber: 10,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 5,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 1,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 4,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 2,
			Transaction:    &types.Transaction{},
		},
	}
	for _, msg := range messages {
		err := els.sequenceExpressLaneSubmission(ctx, msg)
		require.NoError(t, err)
	}
	// We should have only published 2, as we are missing sequence number 3.
	require.Equal(t, 2, len(stubPublisher.publishedTxOrder))
	require.Equal(t, len(messages), len(els.messagesBySequenceNumber))

	err := els.sequenceExpressLaneSubmission(ctx, &timeboost.ExpressLaneSubmission{SequenceNumber: 3, Transaction: &types.Transaction{}})
	require.NoError(t, err)
	require.Equal(t, 5, len(stubPublisher.publishedTxOrder))
}

func Test_expressLaneService_sequenceExpressLaneSubmission_erroredTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:             lru.NewCache[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	stubPublisher := makeStubPublisher(els)
	els.transactionPublisher = stubPublisher

	messages := []*timeboost.ExpressLaneSubmission{
		{
			SequenceNumber: 1,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 3,
			Transaction:    &types.Transaction{},
		},
		{
			SequenceNumber: 2,
			Transaction:    nil,
		},
		{
			SequenceNumber: 2,
			Transaction:    &types.Transaction{},
		},
	}
	for _, msg := range messages {
		if msg.Transaction == nil {
			err := els.sequenceExpressLaneSubmission(ctx, msg)
			require.ErrorContains(t, err, "oops, bad tx")
		} else {
			err := els.sequenceExpressLaneSubmission(ctx, msg)
			require.NoError(t, err)
		}
	}
	// One tx out of the four should have failed, so we should have only published 3.
	require.Equal(t, 3, len(stubPublisher.publishedTxOrder))
	require.Equal(t, []uint64{1, 2, 3}, stubPublisher.publishedTxOrder)
}

func TestIsWithinAuctionCloseWindow(t *testing.T) {
	initialTimestamp := time.Date(2024, 8, 8, 15, 0, 0, 0, time.UTC)
	roundDuration := 1 * time.Minute
	auctionClosing := 15 * time.Second

	es := &expressLaneService{
		initialTimestamp: initialTimestamp,
		roundDuration:    roundDuration,
		auctionClosing:   auctionClosing,
	}

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
			actual := es.isWithinAuctionCloseWindow(tt.arrivalTime)
			if actual != tt.expectedBool {
				t.Errorf("isWithinAuctionCloseWindow(%v) = %v; want %v", tt.arrivalTime, actual, tt.expectedBool)
			}
		})
	}
}

func Benchmark_expressLaneService_validateExpressLaneTx(b *testing.B) {
	b.StopTimer()
	addr := crypto.PubkeyToAddress(testPriv.PublicKey)
	es := &expressLaneService{
		auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
		initialTimestamp:    time.Now(),
		roundDuration:       time.Minute,
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
