package timeboost

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestAuctioneer_validateBid(t *testing.T) {
	tests := []struct {
		name          string
		bid           *Bid
		expectedErr   error
		errMsg        string
		auctionClosed bool
	}{
		{
			name:        "nil bid",
			bid:         nil,
			expectedErr: ErrMalformedData,
			errMsg:      "nil bid",
		},
		{
			name:        "empty express lane controller address",
			bid:         &Bid{},
			expectedErr: ErrMalformedData,
			errMsg:      "empty express lane controller address",
		},
		{
			name: "incorrect chain id",
			bid: &Bid{
				ExpressLaneController: common.Address{'b'},
			},
			expectedErr: ErrWrongChainId,
			errMsg:      "can not auction for chain id: 0",
		},
		{
			name: "incorrect round",
			bid: &Bid{
				ExpressLaneController: common.Address{'b'},
				ChainId:               big.NewInt(1),
			},
			expectedErr: ErrBadRoundNumber,
			errMsg:      "wanted 1, got 0",
		},
		{
			name: "auction is closed",
			bid: &Bid{
				ExpressLaneController: common.Address{'b'},
				ChainId:               big.NewInt(1),
				Round:                 1,
			},
			expectedErr:   ErrBadRoundNumber,
			errMsg:        "auction is closed",
			auctionClosed: true,
		},
		{
			name: "lower than reserved price",
			bid: &Bid{
				ExpressLaneController: common.Address{'b'},
				ChainId:               big.NewInt(1),
				Round:                 1,
				Amount:                big.NewInt(1),
			},
			expectedErr: ErrReservePriceNotMet,
			errMsg:      "reserve price 2, bid 1",
		},
		{
			name: "incorrect signature",
			bid: &Bid{
				ExpressLaneController: common.Address{'b'},
				ChainId:               big.NewInt(1),
				Round:                 1,
				Amount:                big.NewInt(3),
				Signature:             []byte{'a'},
			},
			expectedErr: ErrMalformedData,
			errMsg:      "signature length is not 65",
		},
		{
			name:        "not a depositor",
			bid:         buildValidBid(t),
			expectedErr: ErrNotDepositor,
		},
	}

	setup := setupAuctionTest(t, context.Background())

	for _, tt := range tests {
		a := Auctioneer{
			chainId:                []*big.Int{big.NewInt(1)},
			initialRoundTimestamp:  time.Now().Add(-time.Second),
			reservePrice:           big.NewInt(2),
			roundDuration:          time.Minute,
			auctionClosingDuration: 45 * time.Second,
			auctionContract:        setup.expressLaneAuction,
		}
		if tt.auctionClosed {
			a.roundDuration = 0
		}
		t.Run(tt.name, func(t *testing.T) {
			_, err := a.validateBid(tt.bid)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Contains(t, err.Error(), tt.errMsg)
		})
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

func buildValidBid(t *testing.T) *Bid {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	b := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: common.Address{'c'},
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
		Signature:              []byte{'a'},
	}

	bidValues, err := encodeBidValues(domainValue, b.ChainId, b.AuctionContractAddress, b.Round, b.Amount, b.ExpressLaneController)
	require.NoError(t, err)

	signature, err := buildSignature(privateKey, bidValues)
	require.NoError(t, err)

	b.Signature = signature

	return b
}
