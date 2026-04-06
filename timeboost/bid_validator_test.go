// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/redisutil"
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

func TestBidValidatorAPI_SubmitBid_NilChecks(t *testing.T) {
	t.Parallel()
	api := &BidValidatorAPI{}
	ctx := context.Background()

	tests := []struct {
		name   string
		bid    *JsonBid
		errMsg string
	}{
		{
			name:   "nil bid",
			bid:    nil,
			errMsg: "nil bid",
		},
		{
			name:   "nil chain id",
			bid:    &JsonBid{},
			errMsg: "nil chain id",
		},
		{
			name:   "nil amount",
			bid:    &JsonBid{ChainId: (*hexutil.Big)(big.NewInt(1))},
			errMsg: "nil amount",
		},
		{
			name:   "nil signature",
			bid:    &JsonBid{ChainId: (*hexutil.Big)(big.NewInt(1)), Amount: (*hexutil.Big)(big.NewInt(1))},
			errMsg: "nil signature",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := api.SubmitBid(ctx, tt.bid)
			require.ErrorIs(t, err, ErrMalformedData)
			require.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestBidValidator_validateBid_failedBidDoesNotConsumeRateLimitSlot(t *testing.T) {
	t.Parallel()
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
	bid.Signature = makeValidSignature(t, err, bidHash, privateKey)

	// Use a balance checker that returns zero to make validation fail after
	// the rate-limit counter has been incremented.
	zeroBalanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(0), nil
	}
	_, err = bv.validateBid(bid, zeroBalanceFn)
	require.ErrorIs(t, err, ErrNotDepositor)

	// The failed bid should not have consumed a rate-limit slot.
	bidder := crypto.PubkeyToAddress(privateKey.PublicKey)
	require.Equal(t, uint8(0), bv.bidCountForSender(bidder), "failed bid should not consume a rate-limit slot")

	// Verify the bidder can still submit the full maxBidsPerSenderInRound.
	validBalanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(10), nil
	}
	for i := 0; i < int(bv.maxBidsPerSenderInRound); i++ {
		_, err := bv.validateBid(bid, validBalanceFn)
		require.NoError(t, err)
	}
	_, err = bv.validateBid(bid, validBalanceFn)
	require.ErrorIs(t, err, ErrTooManyBids)
}

func TestBidValidator_validateBid_rollbackDoesNotUnderflowAfterRoundReset(t *testing.T) {
	t.Parallel()
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
	bidder := crypto.PubkeyToAddress(privateKey.PublicKey)
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
	bid.Signature = makeValidSignature(t, err, bidHash, privateKey)

	// Simulate a round reset clearing the map while the bid is being validated.
	// The balance checker runs between the rate-limit increment and the deferred
	// rollback, so we use it to clear the map (as the round-clearing goroutine would).
	roundResetBalanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		bv.mu.Lock()
		bv.bidsPerSenderInRound = make(map[common.Address]uint8)
		bv.mu.Unlock()
		return big.NewInt(0), nil // Return zero to trigger ErrNotDepositor and the rollback.
	}
	_, err = bv.validateBid(bid, roundResetBalanceFn)
	require.ErrorIs(t, err, ErrNotDepositor)

	// The rollback's count > 0 guard must prevent uint8 underflow (0 - 1 = 255).
	require.Equal(t, uint8(0), bv.bidCountForSender(bidder), "rate-limit count must remain 0 after round reset, not underflow to 255")
}

func TestBidValidator_validateBid_concurrentRollbackCorrectness(t *testing.T) {
	t.Parallel()
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
		maxBidsPerSenderInRound:        255, // high limit so rate-limiting doesn't interfere
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: common.Hash{},
	}
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	bidder := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Build a template bid and sign it once.
	templateBid := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: auctionContractAddr,
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
		Signature:              []byte{'a'},
	}
	bidHash, err := templateBid.ToEIP712Hash(bv.auctionContractDomainSeparator)
	require.NoError(t, err)
	sig := makeValidSignature(t, err, bidHash, privateKey)

	const concurrency = 10
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	for i := 0; i < concurrency; i++ {
		i := i // capture loop variable
		wg.Add(1)
		// Each goroutine gets its own Bid copy so they don't share *big.Int
		// pointers, which the EIP-712 hashing code may mutate internally.
		bidCopy := &Bid{
			ExpressLaneController:  templateBid.ExpressLaneController,
			AuctionContractAddress: templateBid.AuctionContractAddress,
			ChainId:                new(big.Int).Set(templateBid.ChainId),
			Round:                  templateBid.Round,
			Amount:                 new(big.Int).Set(templateBid.Amount),
			Signature:              append([]byte(nil), sig...),
		}
		balanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
			// Alternate: even goroutines succeed, odd goroutines fail
			if i%2 == 0 {
				return big.NewInt(10), nil
			}
			return big.NewInt(0), nil
		}
		go func() {
			defer wg.Done()
			_, err := bv.validateBid(bidCopy, balanceFn)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failCount, 1)
			}
		}()
	}
	wg.Wait()

	// The final count should equal exactly the number of successful validations.
	count := bv.bidCountForSender(bidder)
	require.Equal(t, successCount, int64(count),
		"rate-limit count must equal successful validations (got %d successes, count=%d)", successCount, count)
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

	// crypto.Sign returns V as 0 or 1 (recovery ID). EIP-712 expects V as
	// 27 or 28, so we add 27 rather than unconditionally setting to 27.
	signature[len(signature)-1] += 27
	return signature
}

func TestBidValidatorAPI_SubmitBid_ProduceFailureRollsBackRateLimit(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testSetup := setupAuctionTest(t, ctx)
	redisURL := redisutil.CreateTestRedis(ctx, t)
	bv, endpoint := setupBidValidator(t, ctx, redisURL, testSetup)
	bc := setupBidderClient(t, ctx, testSetup.accounts[0], testSetup, endpoint)
	require.NoError(t, bc.Deposit(ctx, big.NewInt(5)))

	// Create a valid bid (this also submits it via RPC, incrementing the rate-limit counter).
	bid, err := bc.Bid(ctx, big.NewInt(5), testSetup.accounts[0].txOpts.From)
	require.NoError(t, err)

	bidder := testSetup.accounts[0].txOpts.From

	// After the successful bid, the rate-limit counter should be 1.
	require.Equal(t, uint8(1), bv.bidCountForSender(bidder))

	// Create a cancelled context so Produce will fail (validateBid does not use
	// the SubmitBid context, so validation will succeed).
	cancelledCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()

	api := &BidValidatorAPI{BidValidator: bv}
	err = api.SubmitBid(cancelledCtx, bid.ToJson())
	require.Error(t, err, "Produce should fail with cancelled context")

	// The rate-limit counter should still be 1 (the failed Produce should have rolled back).
	require.Equal(t, uint8(1), bv.bidCountForSender(bidder), "rate-limit counter should be rolled back after Produce failure")
}

func TestBidValidator_SetReservePriceCopiesValue(t *testing.T) {
	t.Parallel()
	bv := BidValidator{
		reservePrice: big.NewInt(0),
	}

	original := big.NewInt(100)
	bv.SetReservePrice(original)

	// Mutating the original should not affect the stored reserve price.
	original.SetInt64(999)

	stored := bv.fetchReservePrice()
	require.Equal(t, big.NewInt(100), stored, "SetReservePrice must copy the value; mutating the original should not affect stored price")
}

// Test 1: Sign → validateBid → recovered bidder must match the signing key.
// Exercises BOTH recovery IDs (V=27 and V=28) deterministically.
func TestBidValidator_validateBid_identityRoundTrip(t *testing.T) {
	t.Parallel()
	auctionContractAddr := common.Address{'a'}
	domainSep := common.Hash{}

	validBalanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(10), nil
	}

	// Run enough iterations to guarantee we see both V=0 and V=1 from crypto.Sign.
	var sawV0, sawV1 bool
	for iter := 0; iter < 50; iter++ {
		bv := BidValidator{
			chainId: big.NewInt(1),
			roundTimingInfo: RoundTimingInfo{
				Offset:         time.Now().Add(-time.Second),
				Round:          time.Minute,
				AuctionClosing: 45 * time.Second,
			},
			reservePrice:                   big.NewInt(2),
			bidsPerSenderInRound:           make(map[common.Address]uint8),
			maxBidsPerSenderInRound:        255,
			auctionContractAddr:            auctionContractAddr,
			auctionContractDomainSeparator: domainSep,
		}
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		expectedBidder := crypto.PubkeyToAddress(privateKey.PublicKey)

		bid := &Bid{
			ExpressLaneController:  common.Address{'b'},
			AuctionContractAddress: auctionContractAddr,
			ChainId:                big.NewInt(1),
			Round:                  1,
			Amount:                 big.NewInt(3),
			Signature:              []byte{'a'},
		}
		bidHash, err := bid.ToEIP712Hash(domainSep)
		require.NoError(t, err)
		sig := makeValidSignature(t, err, bidHash, privateKey)

		// Track which recovery IDs we see.
		v := sig[64]
		if v == 27 {
			sawV0 = true
		} else if v == 28 {
			sawV1 = true
		} else {
			t.Fatalf("unexpected V value %d", v)
		}

		bid.Signature = sig

		validated, err := bv.validateBid(bid, validBalanceFn)
		require.NoError(t, err, "iter %d: validateBid must succeed", iter)
		require.Equal(t, expectedBidder, validated.Bidder,
			"iter %d: recovered bidder must match signing key (V=%d)", iter, v)
	}
	require.True(t, sawV0 && sawV1, "expected to see both V=27 and V=28 across iterations (saw V0=%v, V1=%v)", sawV0, sawV1)
}

// Test 2: EIP-712 hash must change when any bid field changes.
func TestToEIP712Hash_fieldSensitivity(t *testing.T) {
	t.Parallel()
	domainSep := common.Hash{}
	base := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: common.Address{'a'},
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
	}
	baseHash, err := base.ToEIP712Hash(domainSep)
	require.NoError(t, err)

	// Same inputs → same hash (determinism).
	clone := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: common.Address{'a'},
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
	}
	cloneHash, err := clone.ToEIP712Hash(domainSep)
	require.NoError(t, err)
	require.Equal(t, baseHash, cloneHash, "identical bids must produce identical hashes")

	// Change round.
	diffRound := &Bid{ExpressLaneController: base.ExpressLaneController, AuctionContractAddress: base.AuctionContractAddress, ChainId: big.NewInt(1), Round: 2, Amount: big.NewInt(3)}
	h, err := diffRound.ToEIP712Hash(domainSep)
	require.NoError(t, err)
	require.NotEqual(t, baseHash, h, "different round must produce different hash")

	// Change amount.
	diffAmt := &Bid{ExpressLaneController: base.ExpressLaneController, AuctionContractAddress: base.AuctionContractAddress, ChainId: big.NewInt(1), Round: 1, Amount: big.NewInt(99)}
	h, err = diffAmt.ToEIP712Hash(domainSep)
	require.NoError(t, err)
	require.NotEqual(t, baseHash, h, "different amount must produce different hash")

	// Change controller.
	diffCtrl := &Bid{ExpressLaneController: common.Address{'z'}, AuctionContractAddress: base.AuctionContractAddress, ChainId: big.NewInt(1), Round: 1, Amount: big.NewInt(3)}
	h, err = diffCtrl.ToEIP712Hash(domainSep)
	require.NoError(t, err)
	require.NotEqual(t, baseHash, h, "different controller must produce different hash")

	// Different domain separator.
	altDomain := common.Hash{0xff}
	h, err = base.ToEIP712Hash(altDomain)
	require.NoError(t, err)
	require.NotEqual(t, baseHash, h, "different domain separator must produce different hash")
}

// Test 4: maxBidsPerSenderInRound enforced under concurrent flood.
func TestBidValidator_validateBid_concurrentRateLimitEnforcement(t *testing.T) {
	t.Parallel()
	auctionContractAddr := common.Address{'a'}
	const maxBids = 5
	bv := BidValidator{
		chainId: big.NewInt(1),
		roundTimingInfo: RoundTimingInfo{
			Offset:         time.Now().Add(-time.Second),
			Round:          time.Minute,
			AuctionClosing: 45 * time.Second,
		},
		reservePrice:                   big.NewInt(2),
		bidsPerSenderInRound:           make(map[common.Address]uint8),
		maxBidsPerSenderInRound:        maxBids,
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: common.Hash{},
	}
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	bidder := crypto.PubkeyToAddress(privateKey.PublicKey)

	templateBid := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: auctionContractAddr,
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
		Signature:              []byte{'a'},
	}
	bidHash, err := templateBid.ToEIP712Hash(bv.auctionContractDomainSeparator)
	require.NoError(t, err)
	sig := makeValidSignature(t, err, bidHash, privateKey)

	validBalanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
		return big.NewInt(10), nil
	}

	const goroutines = 30
	var wg sync.WaitGroup
	var successCount int64
	var rateLimitedCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		bidCopy := &Bid{
			ExpressLaneController:  templateBid.ExpressLaneController,
			AuctionContractAddress: templateBid.AuctionContractAddress,
			ChainId:                new(big.Int).Set(templateBid.ChainId),
			Round:                  templateBid.Round,
			Amount:                 new(big.Int).Set(templateBid.Amount),
			Signature:              append([]byte(nil), sig...),
		}
		go func() {
			defer wg.Done()
			_, err := bv.validateBid(bidCopy, validBalanceFn)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, ErrTooManyBids) {
				atomic.AddInt64(&rateLimitedCount, 1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(maxBids), successCount, "exactly maxBidsPerSenderInRound bids must succeed")
	require.Equal(t, int64(goroutines-maxBids), rateLimitedCount, "remaining bids must be rate-limited")
	require.Equal(t, uint8(maxBids), bv.bidCountForSender(bidder), "final count must equal maxBidsPerSenderInRound")
}

// Test 6: Concurrent validations with a round reset mid-flight.
func TestBidValidator_validateBid_concurrentRoundReset(t *testing.T) {
	t.Parallel()
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
		maxBidsPerSenderInRound:        255,
		auctionContractAddr:            auctionContractAddr,
		auctionContractDomainSeparator: common.Hash{},
	}
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	templateBid := &Bid{
		ExpressLaneController:  common.Address{'b'},
		AuctionContractAddress: auctionContractAddr,
		ChainId:                big.NewInt(1),
		Round:                  1,
		Amount:                 big.NewInt(3),
		Signature:              []byte{'a'},
	}
	bidHash, err := templateBid.ToEIP712Hash(bv.auctionContractDomainSeparator)
	require.NoError(t, err)
	sig := makeValidSignature(t, err, bidHash, privateKey)

	const goroutines = 20
	var wg sync.WaitGroup

	// Half the goroutines will have their balance check trigger a round reset.
	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		bidCopy := &Bid{
			ExpressLaneController:  templateBid.ExpressLaneController,
			AuctionContractAddress: templateBid.AuctionContractAddress,
			ChainId:                new(big.Int).Set(templateBid.ChainId),
			Round:                  templateBid.Round,
			Amount:                 new(big.Int).Set(templateBid.Amount),
			Signature:              append([]byte(nil), sig...),
		}
		balanceFn := func(_ *bind.CallOpts, _ common.Address) (*big.Int, error) {
			if i%3 == 0 {
				// Simulate round reset during validation.
				bv.mu.Lock()
				bv.bidsPerSenderInRound = make(map[common.Address]uint8)
				bv.mu.Unlock()
				return big.NewInt(0), nil // Will fail → trigger rollback.
			}
			return big.NewInt(10), nil
		}
		go func() {
			defer wg.Done()
			bv.validateBid(bidCopy, balanceFn)
		}()
	}
	wg.Wait()

	// The key assertion: no panic, no underflow.
	// After round resets, the count must be <= 255 (not wrapped around to 255 via underflow).
	bv.mu.RLock()
	for _, count := range bv.bidsPerSenderInRound {
		require.True(t, count < 200, "count %d suggests underflow", count)
	}
	bv.mu.RUnlock()
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
