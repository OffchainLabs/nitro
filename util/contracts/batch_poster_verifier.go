// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package contracts

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type BatchPosterVerifier struct {
	seqInboxCaller *bridgegen.SequencerInboxCaller
	cache          map[common.Address]bool
	cacheExpiry    time.Time
	mutex          sync.Mutex
}

// Note that we only cache positive instances, not negative ones. That's because we're willing to accept the
// consequences of a false positive (accepting a Store from a recently retired batch poster), but we don't want
// to accept the consequences of a false negative (rejecting a Store from a recently added batch poster).

var batchPosterVerifierLifetime = time.Hour

func NewBatchPosterVerifier(seqInboxCaller *bridgegen.SequencerInboxCaller) *BatchPosterVerifier {
	return &BatchPosterVerifier{
		seqInboxCaller: seqInboxCaller,
		cache:          make(map[common.Address]bool),
		cacheExpiry:    time.Now().Add(batchPosterVerifierLifetime),
	}
}

func (bpv *BatchPosterVerifier) IsBatchPoster(ctx context.Context, addr common.Address) (bool, error) {
	bpv.mutex.Lock()
	if time.Now().After(bpv.cacheExpiry) {
		if err := bpv.flushCache_locked(ctx); err != nil {
			bpv.mutex.Unlock()
			return false, err
		}
	}
	if bpv.cache[addr] {
		bpv.mutex.Unlock()
		return true, nil
	}
	bpv.mutex.Unlock()

	isBatchPoster, err := bpv.seqInboxCaller.IsBatchPoster(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return false, err
	}
	if isBatchPoster {
		bpv.mutex.Lock()
		bpv.cache[addr] = true
		bpv.mutex.Unlock()
	}
	return isBatchPoster, nil
}

func (bpv *BatchPosterVerifier) FlushCache(ctx context.Context) error {
	bpv.mutex.Lock()
	defer bpv.mutex.Unlock()
	return bpv.flushCache_locked(ctx)
}

func (bpv *BatchPosterVerifier) flushCache_locked(ctx context.Context) error {
	bpv.cache = make(map[common.Address]bool)
	bpv.cacheExpiry = time.Now().Add(batchPosterVerifierLifetime)
	return nil
}

func NewMockBatchPosterVerifier(validAddr common.Address) *MockBatchPosterVerifier {
	return &MockBatchPosterVerifier{
		validAddr: validAddr,
	}
}

type MockBatchPosterVerifier struct {
	validAddr common.Address
}

func (bpv *MockBatchPosterVerifier) IsBatchPoster(_ context.Context, addr common.Address) (bool, error) {
	return addr == bpv.validAddr, nil
}

type BatchPosterVerifierInterface interface {
	IsBatchPoster(ctx context.Context, addr common.Address) (bool, error)
}
