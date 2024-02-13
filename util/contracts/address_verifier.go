// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	seqInboxCaller *bridgegen.SequencerInboxCaller
	cache          map[common.Address]bool
	cacheExpiry    time.Time
	mutex          sync.Mutex
}

// Note that we only cache positive instances, not negative ones. That's because we're willing to accept the
// consequences of a false positive (accepting a Store from a recently retired batch poster), but we don't want
// to accept the consequences of a false negative (rejecting a Store from a recently added batch poster).

var addressVerifierLifetime = time.Hour

func NewAddressVerifier(seqInboxCaller *bridgegen.SequencerInboxCaller) *AddressVerifier {
	return &AddressVerifier{
		seqInboxCaller: seqInboxCaller,
		cache:          make(map[common.Address]bool),
		cacheExpiry:    time.Now().Add(addressVerifierLifetime),
	}
}

func (av *AddressVerifier) IsBatchPosterOrSequencer(ctx context.Context, addr common.Address) (bool, error) {
	av.mutex.Lock()
	if time.Now().After(av.cacheExpiry) {
		if err := av.flushCache_locked(ctx); err != nil {
			av.mutex.Unlock()
			return false, err
		}
	}
	if av.cache[addr] {
		av.mutex.Unlock()
		return true, nil
	}
	av.mutex.Unlock()

	result, err := av.seqInboxCaller.IsBatchPoster(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return false, err
	}
	if !result {
		var err error
		result, err = av.seqInboxCaller.IsSequencer(&bind.CallOpts{Context: ctx}, addr)
		if err != nil {
			return false, err
		}
	}
	if result {
		av.mutex.Lock()
		av.cache[addr] = true
		av.mutex.Unlock()
		return true, nil
	}
	return result, nil
}

func (av *AddressVerifier) FlushCache(ctx context.Context) error {
	av.mutex.Lock()
	defer av.mutex.Unlock()
	return av.flushCache_locked(ctx)
}

func (av *AddressVerifier) flushCache_locked(ctx context.Context) error {
	av.cache = make(map[common.Address]bool)
	av.cacheExpiry = time.Now().Add(addressVerifierLifetime)
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

func (bpv *MockAddressVerifier) IsBatchPosterOrSequencer(_ context.Context, addr common.Address) (bool, error) {
	return addr == bpv.validAddr, nil
}

type AddressVerifierInterface interface {
	IsBatchPosterOrSequencer(ctx context.Context, addr common.Address) (bool, error)
}
