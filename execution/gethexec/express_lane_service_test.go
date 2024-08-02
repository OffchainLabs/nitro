package gethexec

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/stretchr/testify/require"
)

var testPriv *ecdsa.PrivateKey

func init() {
	privKey, err := crypto.HexToECDSA("93be75cc4df7acbb636b6abe6de2c0446235ac1dc7da9f290a70d83f088b486d")
	if err != nil {
		panic(err)
	}
	testPriv = privKey
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil tx",
			sub:  &timeboost.ExpressLaneSubmission{},
			es: &expressLaneService{
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil sig",
			sub: &timeboost.ExpressLaneSubmission{
				Transaction: &types.Transaction{},
			},
			es: &expressLaneService{
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "wrong chain id",
			es: &expressLaneService{
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
			expectedErr: timeboost.ErrNoOnchainController,
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: common.Address{'b'},
			},
			sub:         buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv),
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
				roundControl: lru.NewBasicLRU[uint64, *expressLaneControl](8),
			},
			control: expressLaneControl{
				controller: crypto.PubkeyToAddress(testPriv.PublicKey),
			},
			sub:   buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv),
			valid: true,
		},
	}

	for _, tt := range tests {
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

func Test_expressLaneService_sequenceExpressLaneSubmission_nonceTooLow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
		roundControl:             lru.NewBasicLRU[uint64, *expressLaneControl](8),
	}
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	msg := &timeboost.ExpressLaneSubmission{
		Sequence: 0,
	}
	publishFn := func(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, delay bool) error {
		return nil
	}
	err := els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
	require.ErrorIs(t, err, timeboost.ErrSequenceNumberTooLow)
}

func Test_expressLaneService_sequenceExpressLaneSubmission_duplicateNonce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:             lru.NewBasicLRU[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	msg := &timeboost.ExpressLaneSubmission{
		Sequence: 2,
	}
	numPublished := 0
	publishFn := func(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, delay bool) error {
		numPublished += 1
		return nil
	}
	err := els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
	require.NoError(t, err)
	// Because the message is for a future sequence number, it
	// should get queued, but not yet published.
	require.Equal(t, 0, numPublished)
	// Sending it again should give us an error.
	err = els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
	require.ErrorIs(t, err, timeboost.ErrDuplicateSequenceNumber)
}

func Test_expressLaneService_sequenceExpressLaneSubmission_outOfOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:             lru.NewBasicLRU[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	numPublished := 0
	publishedTxOrder := make([]uint64, 0)
	control, _ := els.roundControl.Get(0)
	publishFn := func(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, delay bool) error {
		numPublished += 1
		publishedTxOrder = append(publishedTxOrder, control.sequence)
		return nil
	}
	messages := []*timeboost.ExpressLaneSubmission{
		{
			Sequence: 10,
		},
		{
			Sequence: 5,
		},
		{
			Sequence: 1,
		},
		{
			Sequence: 4,
		},
		{
			Sequence: 2,
		},
	}
	for _, msg := range messages {
		err := els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
		require.NoError(t, err)
	}
	// We should have only published 2, as we are missing sequence number 3.
	require.Equal(t, 2, numPublished)
	require.Equal(t, len(messages), len(els.messagesBySequenceNumber))
}

func Test_expressLaneService_sequenceExpressLaneSubmission_erroredTx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	els := &expressLaneService{
		roundControl:             lru.NewBasicLRU[uint64, *expressLaneControl](8),
		messagesBySequenceNumber: make(map[uint64]*timeboost.ExpressLaneSubmission),
	}
	els.roundControl.Add(0, &expressLaneControl{
		sequence: 1,
	})
	numPublished := 0
	publishedTxOrder := make([]uint64, 0)
	control, _ := els.roundControl.Get(0)
	publishFn := func(parentCtx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions, delay bool) error {
		if tx == nil {
			return errors.New("oops, bad tx")
		}
		numPublished += 1
		publishedTxOrder = append(publishedTxOrder, control.sequence)
		return nil
	}
	messages := []*timeboost.ExpressLaneSubmission{
		{
			Sequence:    1,
			Transaction: &types.Transaction{},
		},
		{
			Sequence:    3,
			Transaction: &types.Transaction{},
		},
		{
			Sequence:    2,
			Transaction: nil,
		},
		{
			Sequence:    2,
			Transaction: &types.Transaction{},
		},
	}
	for _, msg := range messages {
		if msg.Transaction == nil {
			err := els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
			require.ErrorContains(t, err, "oops, bad tx")
		} else {
			err := els.sequenceExpressLaneSubmission(ctx, msg, publishFn)
			require.NoError(t, err)
		}
	}
	// One tx out of the four should have failed, so we should have only published 3.
	require.Equal(t, 3, numPublished)
	require.Equal(t, []uint64{1, 2, 3}, publishedTxOrder)
}

func Benchmark_expressLaneService_validateExpressLaneTx(b *testing.B) {
	b.StopTimer()
	addr := crypto.PubkeyToAddress(testPriv.PublicKey)
	es := &expressLaneService{
		auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
		initialTimestamp:    time.Now(),
		roundDuration:       time.Minute,
		roundControl:        lru.NewBasicLRU[uint64, *expressLaneControl](8),
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
	}
	es.roundControl.Add(0, &expressLaneControl{
		sequence:   1,
		controller: addr,
	})
	sub := buildValidSubmission(b, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), testPriv)
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
) *timeboost.ExpressLaneSubmission {
	b := &timeboost.ExpressLaneSubmission{
		ChainId:                big.NewInt(1),
		AuctionContractAddress: auctionContractAddr,
		Transaction:            types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
		Signature:              make([]byte, 65),
		Round:                  0,
	}
	data, err := b.ToMessageBytes()
	require.NoError(t, err)
	signature, err := buildSignature(privKey, data)
	require.NoError(t, err)
	b.Signature = signature
	return b
}
