// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

type ChainFetchDAS struct {
	DataAvailabilityService
	seqInboxCaller   *bridgegen.SequencerInboxCaller
	seqInboxFilterer *bridgegen.SequencerInboxFilterer
	keysetCache      map[[32]byte][]byte
}

func NewChainFetchDAS(inner DataAvailabilityService, l1client arbutil.L1Interface, seqInboxAddr common.Address) (*ChainFetchDAS, error) {
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	if err != nil {
		return nil, err
	}

	return &ChainFetchDAS{
		inner,
		&seqInbox.SequencerInboxCaller,
		&seqInbox.SequencerInboxFilterer,
		make(map[[32]byte][]byte),
	}, nil
}

func (daReader *ChainFetchDAS) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	// try to fetch from the cache
	var hash32 [32]byte
	copy(hash32[:], hash)
	res, ok := daReader.keysetCache[hash32]
	if ok {
		return res, nil
	}

	// try to fetch from the inner DAS
	innerRes, err := daReader.DataAvailabilityService.GetByHash(ctx, hash)
	if err == nil && bytes.Equal(hash, crypto.Keccak256(innerRes)) {
		daReader.keysetCache[hash32] = innerRes
		return innerRes, nil
	}

	// try to fetch from the L1 chain
	blockNumBig, err := daReader.seqInboxCaller.GetKeysetCreationBlock(&bind.CallOpts{Context: ctx}, hash32)
	if err != nil {
		return nil, err
	}
	if !blockNumBig.IsUint64() {
		return nil, errors.New("block number too large")
	}
	blockNum := blockNumBig.Uint64()
	blockNumPlus1 := blockNum + 1

	filterOpts := &bind.FilterOpts{
		Start:   blockNum,
		End:     &blockNumPlus1,
		Context: ctx,
	}
	iter, err := daReader.seqInboxFilterer.FilterSetValidKeyset(filterOpts, [][32]byte{hash32})
	if err != nil {
		return nil, err
	}
	for iter.Next() {
		if bytes.Equal(hash, crypto.Keccak256(iter.Event.KeysetBytes)) {
			daReader.keysetCache[hash32] = iter.Event.KeysetBytes
			return iter.Event.KeysetBytes, nil
		}
	}
	if iter.Error() != nil {
		return nil, iter.Error()
	}

	return nil, errors.New("Keyset not found")
}
