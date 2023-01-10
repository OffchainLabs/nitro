// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package contracts

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type AddressVerifier struct {
	seqInboxCaller         *bridgegen.SequencerInboxCaller
	batchPosterCache       map[common.Address]bool
	batchPosterCacheExpiry time.Time
	mutex                  sync.Mutex
}

// Note that we only batchPosterCache positive instances, not negative ones. That's because we're willing to accept the
// consequences of a false positive (accepting a Store from a recently retired batch poster), but we don't want
// to accept the consequences of a false negative (rejecting a Store from a recently added batch poster).

var addressVerifierLifetime = time.Hour

func NewAddressVerifier(seqInboxCaller *bridgegen.SequencerInboxCaller) *AddressVerifier {
	return &AddressVerifier{
		seqInboxCaller:         seqInboxCaller,
		batchPosterCache:       make(map[common.Address]bool),
		batchPosterCacheExpiry: time.Now().Add(addressVerifierLifetime),
	}
}

func (bpv *AddressVerifier) IsBatchPoster(ctx context.Context, addr common.Address) (bool, error) {
	bpv.mutex.Lock()
	if time.Now().After(bpv.batchPosterCacheExpiry) {
		if err := bpv.flushCache_locked(ctx); err != nil {
			bpv.mutex.Unlock()
			return false, err
		}
	}
	if bpv.batchPosterCache[addr] {
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
		bpv.batchPosterCache[addr] = true
		bpv.mutex.Unlock()
	}
	return isBatchPoster, nil
}

func (bpv *AddressVerifier) FlushCache(ctx context.Context) error {
	bpv.mutex.Lock()
	defer bpv.mutex.Unlock()
	return bpv.flushCache_locked(ctx)
}

func (bpv *AddressVerifier) flushCache_locked(ctx context.Context) error {
	bpv.batchPosterCache = make(map[common.Address]bool)
	bpv.batchPosterCacheExpiry = time.Now().Add(addressVerifierLifetime)
	return nil
}

func NewMockAddressVerifier(validAddr common.Address) *MockAddressVerifier {
	return &MockAddressVerifier{
		validAddr: validAddr,
	}
}

type MockAddressVerifier struct {
	validAddr common.Address
}

func (bpv *MockAddressVerifier) IsBatchPoster(_ context.Context, addr common.Address) (bool, error) {
	return addr == bpv.validAddr, nil
}

type AddressVerifierInterface interface {
	IsBatchPoster(ctx context.Context, addr common.Address) (bool, error)
}
