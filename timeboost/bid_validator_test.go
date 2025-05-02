package timeboost

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestBidValidator_validateBid(t *testing.T) {
	t.Parallel()
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
		bv := BidValidator{
			chainId: big.NewInt(1),
			roundTimingInfo: RoundTimingInfo{
				Offset:         time.Now().Add(-time.Second * 3),
				Round:          10 * time.Second,
				AuctionClosing: 5 * time.Second,
			},
			reservePrice:            big.NewInt(2),
			auctionContract:         setup.expressLaneAuction,
			auctionContractAddr:     setup.expressLaneAuctionAddr,
			bidsPerSenderInRound:    make(map[common.Address]uint8),
			maxBidsPerSenderInRound: 5,
		}
		t.Run(tt.name, func(t *testing.T) {
			if tt.auctionClosed {
				time.Sleep(time.Second * 3)
			}
			_, err := bv.validateBid(tt.bid, setup.expressLaneAuction.BalanceOf)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestBidValidator_validateBid_perRoundBidLimitReached(t *testing.T) {
	t.Parallel()
	balanceCheckerFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(10), nil
	}
	auctionContractAddr := common.Address{'a'}
	bv := BidValidator{
		chainId: big.NewInt(1),
		roundTimingInfo: RoundTimingInfo{
			Offset:         time.Now().Add(-time.Second),
			Round:          time.Minute,
			AuctionClosing: 45 * time.Second,
		},
		reservePrice:                   big.NewInt(2),
		bidsPerSenderInRound:           make(map[common.Address]uint8),
		maxBidsPerSenderInRound:        5,
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: common.Hash{},
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

	bidHash, err := bid.ToEIP712Hash(bv.auctionContractDomainSeparator)
	require.NoError(t, err)

	signature := makeValidSignature(t, err, bidHash, privateKey)
	bid.Signature = signature
	for i := 0; i < int(bv.maxBidsPerSenderInRound); i++ {
		_, err := bv.validateBid(bid, balanceCheckerFn)
		require.NoError(t, err)
	}
	_, err = bv.validateBid(bid, balanceCheckerFn)
	require.ErrorIs(t, err, ErrTooManyBids)

}

func makeValidSignature(t *testing.T, err error, bidHash common.Hash, privateKey *ecdsa.PrivateKey) []byte {
	signature, err := crypto.Sign(bidHash[:], privateKey)
	require.NoError(t, err)

	signature[len(signature)-1] = 27
	return signature
}

func buildValidBid(t *testing.T, auctionContractAddr common.Address) *Bid {
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

	bidHash, err := bid.ToEIP712Hash(common.Hash{})
	require.NoError(t, err)

	bid.Signature = makeValidSignature(t, err, bidHash, privateKey)

	return bid
}
