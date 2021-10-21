//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"github.com/offchainlabs/arbstate/arbos/queue"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)


type ArbosState struct {
	formatVersion   *big.Int
	nextAlloc       *common.Hash
	gasPool         *storage.StorageBackedInt64
	smallGasPool    *storage.StorageBackedInt64
	gasPriceWei     *big.Int
	l1PricingState  *L1PricingState
	retryableQueue  *queue.QueueInStorage
	validRetryables *storage.Storage
	timestamp       *uint64
	backingStorage  *storage.Storage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := storage.NewGeth(stateDB)

	for tryStorageUpgrade(backingStorage) {
	}

	return &ArbosState{
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		backingStorage,
	}
}

func tryStorageUpgrade(backingStorage *storage.Storage) bool {
	formatVersion := backingStorage.Get(util.IntToHash(0))
	switch formatVersion {
	case util.IntToHash(0):
		upgrade_0_to_1(backingStorage)
		return true
	default:
		return false
	}
}

var (
	versionKey       = util.IntToHash(0)
	storageOffsetKey = util.IntToHash(1)
	gasPoolKey       = util.IntToHash(2)
	smallGasPoolKey  = util.IntToHash(3)
	gasPriceKey      = util.IntToHash(4)
	retryableQueueKey = util.IntToHash(5)
	l1PricingKey     = util.IntToHash(6)
	timestampKey     = util.IntToHash(7)
	validRetryableSetUniqueKey = common.BytesToHash(crypto.Keccak256([]byte("Arbitrum ArbOS valid retryable set unique key")))
)

func upgrade_0_to_1(backingStorage *storage.Storage) {
	backingStorage.Set(versionKey, util.IntToHash(1))
	backingStorage.Set(storageOffsetKey, crypto.Keccak256Hash([]byte("Arbitrum ArbOS storage allocation start point")))
	backingStorage.Set(gasPoolKey, util.IntToHash(GasPoolMax))
	backingStorage.Set(smallGasPoolKey, util.IntToHash(SmallGasPoolMax))
	backingStorage.Set(gasPriceKey, util.IntToHash(1000000000)) // 1 gwei
	backingStorage.Set(l1PricingKey, util.IntToHash(0))
	backingStorage.Set(timestampKey, util.IntToHash(0))
	backingStorage.Set(retryableQueueKey, util.IntToHash(0))
}

func (state *ArbosState) FormatVersion() *big.Int {
	if state.formatVersion == nil {
		state.formatVersion = state.backingStorage.Get(versionKey).Big()
	}
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val *big.Int) {
	state.formatVersion = val
	state.backingStorage.Set(versionKey, common.BigToHash(state.formatVersion))
}

func (state *ArbosState) AllocateEmptyStorageOffset() *common.Hash {
	if state.nextAlloc == nil {
		val := state.backingStorage.Get(storageOffsetKey)
		state.nextAlloc = &val
	}
	ret := state.nextAlloc
	nextAlloc := crypto.Keccak256Hash(state.nextAlloc.Bytes())
	state.nextAlloc = &nextAlloc
	state.backingStorage.Set(storageOffsetKey, nextAlloc)
	return ret
}

func (state *ArbosState) GasPool() int64 {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(gasPoolKey)
	}
	return state.gasPool.Get()
}

func (state *ArbosState) SetGasPool(val int64) {
	if state.gasPool == nil {
		state.gasPool = state.backingStorage.OpenStorageBackedInt64(gasPoolKey)
	}
	state.gasPool.Set(val)
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(smallGasPoolKey)
	}
	return state.smallGasPool.Get()
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	if state.smallGasPool == nil {
		state.smallGasPool = state.backingStorage.OpenStorageBackedInt64(smallGasPoolKey)
	}
	state.smallGasPool.Set(val)
}

func (state *ArbosState) GasPriceWei() *big.Int {
	if state.gasPriceWei == nil {
		state.gasPriceWei = state.backingStorage.Get(gasPriceKey).Big()
	}
	return state.gasPriceWei
}

func (state *ArbosState) SetGasPriceWei(val *big.Int) {
	state.gasPriceWei = val
	state.backingStorage.Set(gasPriceKey, common.BigToHash(val))
}

func (state *ArbosState) RetryableQueue() *queue.QueueInStorage {
	if state.retryableQueue == nil {
		queueOffset := state.backingStorage.Get(retryableQueueKey)
		if queueOffset == util.IntToHash(0) {
			var q *queue.QueueInStorage
			q, queueOffset = queue.AllocateQueueInStorage(state.backingStorage)
			state.retryableQueue = q
			state.backingStorage.Set(retryableQueueKey, queueOffset)
		} else {
			state.retryableQueue = queue.OpenQueueInStorage(state.backingStorage, queueOffset)
		}
	}
	return state.retryableQueue
}

func (state *ArbosState) L1PricingState() *L1PricingState {
	if state.l1PricingState == nil {
		offset := state.backingStorage.Get(l1PricingKey)
		if offset == (common.Hash{}) {
			l1PricingState, offset := AllocateL1PricingState(state)
			state.l1PricingState = l1PricingState
			state.backingStorage.Set(l1PricingKey, offset)
		} else {
			state.l1PricingState = OpenL1PricingState(offset, state.backingStorage)
		}
	}
	return state.l1PricingState
}

func (state *ArbosState) LastTimestampSeen() uint64 {
	if state.timestamp == nil {
		ts := state.backingStorage.Get(timestampKey).Big().Uint64()
		state.timestamp = &ts
	}
	return *state.timestamp
}

func (state *ArbosState) SetLastTimestampSeen(val uint64) {
	if state.timestamp == nil {
		ts := state.backingStorage.Get(timestampKey).Big().Uint64()
		state.timestamp = &ts
	}
	if val < *state.timestamp {
		panic("timestamp decreased")
	}
	if val > *state.timestamp {
		delta := val - *state.timestamp
		ts := val
		state.timestamp = &ts
		state.backingStorage.Set(timestampKey, util.IntToHash(int64(ts)))
		state.notifyGasPricerThatTimeElapsed(delta)
	}
}

func (state *ArbosState) ValidRetryablesSet() *storage.Storage {
	// This is a virtual storage (KVS) that we use to keep track of which ids are ids of valid retryables.
	// We need this because untrusted users will be submitting ids, and we need to check them for validity, so that
	//     we don't treat some maliciously chosen storage of our storage as a valid retryable.
	return state.backingStorage.Open(validRetryableSetUniqueKey.Bytes())
}
