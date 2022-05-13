// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"time"
)

type BatchPosterVerifier struct {
	seqInboxCaller *bridgegen.SequencerInboxCaller
	cache          map[common.Address]bool
	cacheExpiry    time.Time
}

var batchPosterVerifierLifetime = time.Hour

func NewBatchPosterVerifier(seqInboxCaller *bridgegen.SequencerInboxCaller) *BatchPosterVerifier {
	return &BatchPosterVerifier{seqInboxCaller, make(map[common.Address]bool), time.Now()}
}

func (bpv *BatchPosterVerifier) IsBatchPoster(ctx context.Context, addr common.Address) (bool, error) {
	if time.Now().After(bpv.cacheExpiry) {
		bpv.cache = make(map[common.Address]bool)
		bpv.cacheExpiry = time.Now().Add(batchPosterVerifierLifetime)
	}
	if bpv.cache[addr] {
		return true, nil
	}
	isBatchPoster, err := bpv.seqInboxCaller.IsBatchPoster(&bind.CallOpts{Context: ctx}, addr)
	if err != nil {
		return false, err
	}
	if isBatchPoster {
		bpv.cache[addr] = true
	}
	return isBatchPoster, nil
}
