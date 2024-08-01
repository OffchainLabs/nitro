package gethexec

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/stretchr/testify/require"
)

func Test_expressLaneService_validateExpressLaneTx(t *testing.T) {
	privKey, err := crypto.HexToECDSA("93be75cc4df7acbb636b6abe6de2c0446235ac1dc7da9f290a70d83f088b486d")
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	tests := []struct {
		name        string
		es          *expressLaneService
		sub         *timeboost.ExpressLaneSubmission
		expectedErr error
		controller  common.Address
		valid       bool
	}{
		{
			name:        "nil msg",
			sub:         nil,
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name:        "nil tx",
			sub:         &timeboost.ExpressLaneSubmission{},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "nil sig",
			sub: &timeboost.ExpressLaneSubmission{
				Transaction: &types.Transaction{},
			},
			expectedErr: timeboost.ErrMalformedData,
		},
		{
			name: "wrong chain id",
			es: &expressLaneService{
				chainConfig: &params.ChainConfig{
					ChainID: big.NewInt(1),
				},
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
				control: expressLaneControl{
					controller: common.Address{'b'},
				},
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
				control: expressLaneControl{
					controller: common.Address{'b'},
				},
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
				control: expressLaneControl{
					controller: common.Address{'b'},
				},
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
				control: expressLaneControl{
					controller: common.Address{'b'},
				},
			},
			sub:         buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), privKey),
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
				control: expressLaneControl{
					controller: addr,
				},
			},
			sub:   buildValidSubmission(t, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), privKey),
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.es.validateExpressLaneTx(tt.sub)
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func Benchmark_expressLaneService_validateExpressLaneTx(b *testing.B) {
	b.StopTimer()
	privKey, err := crypto.HexToECDSA("93be75cc4df7acbb636b6abe6de2c0446235ac1dc7da9f290a70d83f088b486d")
	require.NoError(b, err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	es := &expressLaneService{
		auctionContractAddr: common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"),
		initialTimestamp:    time.Now(),
		roundDuration:       time.Minute,
		chainConfig: &params.ChainConfig{
			ChainID: big.NewInt(1),
		},
		control: expressLaneControl{
			controller: addr,
		},
	}
	sub := buildValidSubmission(b, common.HexToAddress("0x2Aef36410182881a4b13664a1E079762D7F716e6"), privKey)
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
