package timeboost

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestAuctioneer_validateBid(t *testing.T) {
	setup := setupAuctionTest(t, context.Background())
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
			errMsg:      "incorrect auction contract address",
		},
		{
			name: "incorrect chain id",
			bid: &Bid{
				ExpressLaneController:  common.Address{'b'},
				AuctionContractAddress: setup.expressLaneAuctionAddr,
				ChainId:                big.NewInt(50),
			},
			expectedErr: ErrWrongChainId,
			errMsg:      "can not auction for chain id: 50",
		},
		{
			name: "incorrect round",
			bid: &Bid{
				ExpressLaneController:  common.Address{'b'},
				AuctionContractAddress: setup.expressLaneAuctionAddr,
				ChainId:                big.NewInt(1),
			},
			expectedErr: ErrBadRoundNumber,
			errMsg:      "wanted 1, got 0",
		},
		{
			name: "auction is closed",
			bid: &Bid{
				ExpressLaneController:  common.Address{'b'},
				AuctionContractAddress: setup.expressLaneAuctionAddr,
				ChainId:                big.NewInt(1),
				Round:                  1,
			},
			expectedErr:   ErrBadRoundNumber,
			errMsg:        "auction is closed",
			auctionClosed: true,
		},
		{
			name: "lower than reserved price",
			bid: &Bid{
				ExpressLaneController:  common.Address{'b'},
				AuctionContractAddress: setup.expressLaneAuctionAddr,
				ChainId:                big.NewInt(1),
				Round:                  1,
				Amount:                 big.NewInt(1),
			},
			expectedErr: ErrReservePriceNotMet,
			errMsg:      "reserve price 2, bid 1",
		},
		{
			name: "incorrect signature",
			bid: &Bid{
				ExpressLaneController:  common.Address{'b'},
				AuctionContractAddress: setup.expressLaneAuctionAddr,
				ChainId:                big.NewInt(1),
				Round:                  1,
				Amount:                 big.NewInt(3),
				Signature:              []byte{'a'},
			},
			expectedErr: ErrMalformedData,
			errMsg:      "signature length is not 65",
		},
		{
			name:        "not a depositor",
			bid:         buildValidBid(t, setup.expressLaneAuctionAddr),
			expectedErr: ErrNotDepositor,
		},
	}

	for _, tt := range tests {
		a := Auctioneer{
			chainId:                 []*big.Int{big.NewInt(1)},
			initialRoundTimestamp:   time.Now().Add(-time.Second),
			reservePrice:            big.NewInt(2),
			roundDuration:           time.Minute,
			auctionClosingDuration:  45 * time.Second,
			auctionContract:         setup.expressLaneAuction,
			auctionContractAddr:     setup.expressLaneAuctionAddr,
			bidsPerSenderInRound:    make(map[common.Address]uint8),
			maxBidsPerSenderInRound: 5,
		}
		if tt.auctionClosed {
			a.roundDuration = 0
		}
		t.Run(tt.name, func(t *testing.T) {
			_, err := a.validateBid(tt.bid, setup.expressLaneAuction.BalanceOf, a.fetchReservePrice)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestAuctioneer_validateBid_perRoundBidLimitReached(t *testing.T) {
	balanceCheckerFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(10), nil
	}
	fetchReservePriceFn := func() *big.Int {
		return big.NewInt(0)
	}
	auctionContractAddr := common.Address{'a'}
	a := Auctioneer{
		chainId:                 []*big.Int{big.NewInt(1)},
		initialRoundTimestamp:   time.Now().Add(-time.Second),
		reservePrice:            big.NewInt(2),
		roundDuration:           time.Minute,
		auctionClosingDuration:  45 * time.Second,
		bidsPerSenderInRound:    make(map[common.Address]uint8),
		maxBidsPerSenderInRound: 5,
		auctionContractAddr:     auctionContractAddr,
	}
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	bid := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: auctionContractAddr,
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
		Signature:              []byte{'a'},
	}
	bidValues, err := encodeBidValues(domainValue, bid.ChainId, bid.AuctionContractAddress, bid.Round, bid.Amount, bid.ExpressLaneController)
	require.NoError(t, err)

	signature, err := buildSignature(privateKey, bidValues)
	require.NoError(t, err)

	bid.Signature = signature
	for i := 0; i < int(a.maxBidsPerSenderInRound)-1; i++ {
		_, err := a.validateBid(bid, balanceCheckerFn, fetchReservePriceFn)
		require.NoError(t, err)
	}
	_, err = a.validateBid(bid, balanceCheckerFn, fetchReservePriceFn)
	require.ErrorIs(t, err, ErrTooManyBids)

}

func buildSignature(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	prefixedData := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data))), data...))
	signature, err := crypto.Sign(prefixedData, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func buildValidBid(t *testing.T, auctionContractAddr common.Address) *Bid {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	b := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: auctionContractAddr,
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
