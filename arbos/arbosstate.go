//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

type EvmStorage interface {
	Get(key common.Hash) common.Hash
	Set(key common.Hash, value common.Hash)
	Swap(key common.Hash, value common.Hash) common.Hash
}

type GethEvmStorage struct {
	account common.Address
	db    vm.StateDB
}

// Use a Geth database to create an evm key-value store
func NewGethEvmStorage(statedb vm.StateDB) *GethEvmStorage {
	return &GethEvmStorage{
		account: common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		db:    statedb,
	}
}

// Use Geth's memory-backed database to create an evm key-value store
func NewMemoryBackingEvmStorage() *GethEvmStorage {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	return NewGethEvmStorage(statedb)
}

func (store *GethEvmStorage) Get(key common.Hash) common.Hash {
	return store.db.GetState(store.account, key)
}

func (store *GethEvmStorage) Set(key common.Hash, value common.Hash) {
	store.db.SetState(store.account, key, value)
}

func (store *GethEvmStorage) Swap(key common.Hash, newValue common.Hash) common.Hash {
	oldValue := store.Get(key)
	store.Set(key, newValue)
	return oldValue
}

func IntToHash(val int64) common.Hash {
	return common.BigToHash(big.NewInt(val))
}

func hashPlusInt(x common.Hash, y int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(x.Big(), big.NewInt(y)))   //BUGBUG: BigToHash(x) converts abs(x) to a Hash
}

type ArbosState struct {
	formatVersion     *big.Int
	nextAlloc         *common.Hash
	gasPool           *StorageBackedInt64
	smallGasPool      *StorageBackedInt64
	gasPriceWei       *big.Int
	lastTimestampSeen *big.Int
	retryableQueue	  *QueueInStorage
	backingStorage    EvmStorage
}

func OpenArbosState(stateDB vm.StateDB) *ArbosState {
	backingStorage := NewGethEvmStorage(stateDB)

	for tryStorageUpgrade(backingStorage) {}

	return &ArbosState{
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

func tryStorageUpgrade(backingStorage EvmStorage) bool {
	formatVersion := backingStorage.Get(IntToHash(0))
	switch formatVersion {
	case IntToHash(0):
		upgrade_0_to_1(backingStorage)
		return true
	default:
		return false
	}
}

var (
	versionKey       = IntToHash(0)
	storageOffsetKey = IntToHash(1)
	gasPoolKey = IntToHash(2)
	smallGasPoolKey = IntToHash(3)
	gasPriceKey = IntToHash(4)
	lastTimestampKey= IntToHash(5)
)

func upgrade_0_to_1(backingStorage EvmStorage) {
	backingStorage.Set(versionKey, IntToHash(1))
	backingStorage.Set(storageOffsetKey, crypto.Keccak256Hash([]byte("Arbitrum ArbOS storage allocation start point")))
	backingStorage.Set(gasPoolKey, IntToHash(10000000*10*60))
	backingStorage.Set(smallGasPoolKey, IntToHash(10000000*60))
	backingStorage.Set(gasPriceKey, IntToHash(1000000000)) // 1 gwei
	backingStorage.Set(lastTimestampKey, IntToHash(0))
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
		state.gasPool = OpenStorageBackedInt64(state.backingStorage, gasPoolKey)
	}
	return state.gasPool.Get()
}

func (state *ArbosState) SetGasPool(val int64) {
	if state.gasPool == nil {
		state.gasPool = OpenStorageBackedInt64(state.backingStorage, gasPoolKey)
	}
	state.gasPool.Set(val)
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		state.smallGasPool = OpenStorageBackedInt64(state.backingStorage, smallGasPoolKey)
	}
	return state.smallGasPool.Get()
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	if state.smallGasPool == nil {
		state.smallGasPool = OpenStorageBackedInt64(state.backingStorage, smallGasPoolKey)
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

func (state *ArbosState) LastTimestampSeen() *big.Int {
	if state.lastTimestampSeen == nil {
		state.lastTimestampSeen = state.backingStorage.Get(lastTimestampKey).Big()
	}
	return state.lastTimestampSeen
}

func (state *ArbosState) SetLastTimestampSeen(val *big.Int) {
	state.lastTimestampSeen = val
	state.backingStorage.Set(lastTimestampKey, common.BigToHash(val))
}

func (state *ArbosState) RetryableQueue() *QueueInStorage {
	if state.retryableQueue != nil {
		queueOffset := state.backingStorage.Get(IntToHash(6))
		state.retryableQueue = OpenQueueInStorage(state, queueOffset)
	}
	return state.retryableQueue
}

func (state *ArbosState) AllocateSegment(size uint64) (*StorageSegment, error) {
	if size > MaxSizedSegmentSize {
		return nil, errors.New("requested segment size too large")
	}

	offset := state.AllocateEmptyStorageOffset()

	state.backingStorage.Set(*offset, IntToHash(int64(size)))

	return &StorageSegment{
		*offset,
		size,
		state.backingStorage,
	}, nil
}

func (state *ArbosState) OpenSegment(offset common.Hash) *StorageSegment {
	rawSize := state.backingStorage.Get(offset)
	bigSize := rawSize.Big()
	if !bigSize.IsUint64() {
		panic("not a valid state segment")
	}
	size := bigSize.Uint64()
	if size == 0 {
		panic("state segment invalid or was deleted")
	}
	if size > MaxSizedSegmentSize {
		panic("state segment size invalid")
	}
	return &StorageSegment{
		offset,
		size,
		state.backingStorage,
	}
}

func (state *ArbosState) AllocateSegmentForBytes(buf []byte) *StorageSegment {
	sizeWords := (len(buf)+31) / 32
	seg, err := state.AllocateSegment(uint64(1+sizeWords))
	if err != nil {
		panic(err)
	}
	seg.Set(0, IntToHash(int64(len(buf))))

	offset := uint64(1)
	for len(buf) >= 32 {
		seg.Set(offset, common.BytesToHash(buf[:32]))
		offset += 1
		buf = buf[32:]
	}

	endBuf := [32]byte{}
	for i := 0; i < len(buf); i++ {
		endBuf[i] = buf[i]
	}
	seg.Set(offset, common.BytesToHash(endBuf[:]))
	return seg
}

func (state *ArbosState) AdvanceTimestampToAtLeast(newTimestamp *big.Int) {
	currentTimestamp := state.LastTimestampSeen()
	if newTimestamp.Cmp(currentTimestamp) > 0 {
		state.SetLastTimestampSeen(newTimestamp)
	}
}


// StorageBackedInt64 exists because the conversions between common.Hash and big.Int that is provided by
//     go-ethereum don't handle negative values cleanly.  This class hides that complexity.
type StorageBackedInt64 struct {
	storage EvmStorage
	offset  common.Hash
	cache   *int64
}

func OpenStorageBackedInt64(storage EvmStorage, offset common.Hash) *StorageBackedInt64 {
	return &StorageBackedInt64{ storage, offset, nil }
}

func (sbi *StorageBackedInt64) Get() int64 {
	if sbi.cache == nil {
		raw := sbi.storage.Get(sbi.offset).Big()
		if raw.Bit(255) != 0 {
			raw = new(big.Int).SetBit(raw, 255, 0)
			raw = new(big.Int).Neg(raw)
		}
		if !raw.IsInt64() {
			panic("expected int64 compatible value in storage")
		}
		i := raw.Int64()
		sbi.cache = &i
	}
	return *sbi.cache
}

func (sbi *StorageBackedInt64) Set(value int64) {
	i := value
	sbi.cache = &i
	var bigValue *big.Int
	if value >= 0 {
		bigValue = big.NewInt(value)
	} else {
		bigValue = new(big.Int).SetBit(big.NewInt(-value), 255, 1)
	}
	sbi.storage.Set(sbi.offset, common.BigToHash(bigValue))
}
