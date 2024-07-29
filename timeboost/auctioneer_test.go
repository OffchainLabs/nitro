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
			name:        "Nil bid",
			bid:         nil,
			expectedErr: ErrMalformedData,
			errMsg:      "nil bid",
		},
		{
			name:        "Empty bidder address",
			bid:         &Bid{},
			expectedErr: ErrMalformedData,
			errMsg:      "empty bidder address",
		},
		{
			name:        "Empty express lane controller address",
			bid:         &Bid{Bidder: common.Address{'a'}},
			expectedErr: ErrMalformedData,
			errMsg:      "empty express lane controller address",
		},
		{
			name: "Incorrect chain id",
			bid: &Bid{
				Bidder:                common.Address{'a'},
				ExpressLaneController: common.Address{'b'},
			},
			expectedErr: ErrWrongChainId,
			errMsg:      "can not auction for chain id: 0",
		},
		{
			name: "Incorrect round",
			bid: &Bid{
				Bidder:                common.Address{'a'},
				ExpressLaneController: common.Address{'b'},
				ChainId:               1,
			},
			expectedErr: ErrBadRoundNumber,
			errMsg:      "wanted 1, got 0",
		},
		{
			name: "Auction is closed",
			bid: &Bid{
				Bidder:                common.Address{'a'},
				ExpressLaneController: common.Address{'b'},
				ChainId:               1,
				Round:                 1,
			},
			expectedErr:   ErrBadRoundNumber,
			errMsg:        "auction is closed",
			auctionClosed: true,
		},
		{
			name: "Lower than reserved price",
			bid: &Bid{
				Bidder:                common.Address{'a'},
				ExpressLaneController: common.Address{'b'},
				ChainId:               1,
				Round:                 1,
				Amount:                big.NewInt(1),
			},
			expectedErr: ErrInsufficientBid,
			errMsg:      "reserve price 2, bid 1",
		},
		{
			name: "incorrect signature",
			bid: &Bid{
				Bidder:                common.Address{'a'},
				ExpressLaneController: common.Address{'b'},
				ChainId:               1,
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
			chainId:                []uint64{1},
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
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	bidderAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	b := &Bid{
		Bidder:                 bidderAddress,
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: common.Address{'c'},
		ChainId:                1,
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
