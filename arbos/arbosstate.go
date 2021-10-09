package arbos

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
)


// We use an interface since *state.stateObject is private
type GethStateObject interface {
	GetState(db state.Database, key common.Hash) common.Hash
	SetState(db state.Database, key common.Hash, value common.Hash)
}

type EvmStorage interface {
	Get(key common.Hash) common.Hash
	Set(key common.Hash, value common.Hash)
	Swap(key common.Hash, value common.Hash) common.Hash
	GetAsInt64(key common.Hash) int64
}

type GethEvmStorage struct {
	state GethStateObject
	db    state.Database
}

// Use a Geth database to create an evm key-value store
func NewGethEvmStorage(statedb *state.StateDB) *GethEvmStorage {
	stateObj := statedb.GetOrNewStateObject(common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"))
	return &GethEvmStorage{
		state: stateObj,
		db:    statedb.Database(),
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
	return store.state.GetState(store.db, key)
}

func (store *GethEvmStorage) GetAsInt64(key common.Hash) int64 {
	rawValue := store.Get(key).Big()
	if rawValue.IsInt64() {
		return rawValue.Int64()
	} else {
		panic("expected int64 in backing storage")
	}
}

func (store *GethEvmStorage) Set(key common.Hash, value common.Hash) {
	store.state.SetState(store.db, key, value)
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
	gasPool           *int64
	smallGasPool      *int64
	gasPriceWei       *big.Int
	lastTimestampSeen *big.Int
	backingStorage    EvmStorage
}

func OpenArbosState(stateDB *state.StateDB) *ArbosState {
	backingStorage := NewGethEvmStorage(stateDB)

	for tryStorageUpgrade(backingStorage) {}

	return &ArbosState{
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
		val := state.backingStorage.GetAsInt64(gasPoolKey)
		state.gasPool = &val
	}
	return *state.gasPool
}

func (state *ArbosState) SetGasPool(val int64) {   //BUGBUG: handle negative values correctly in storage read/write
	c := val
	state.gasPool = &c
	state.backingStorage.Set(gasPoolKey, IntToHash(c))
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		val := state.backingStorage.GetAsInt64(smallGasPoolKey)
		state.smallGasPool = &val
	}
	return *state.smallGasPool
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	c := val
	state.smallGasPool = &c
	state.backingStorage.Set(smallGasPoolKey, IntToHash(c))
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


type SizedArbosStorageSegment struct {
	offset      common.Hash
	size        uint64
	storage     *ArbosState
}

const MaxSizedSegmentSize = 1<<48

func (state *ArbosState) AllocateSizedSegment(size uint64) (*SizedArbosStorageSegment, error) {
	if size > MaxSizedSegmentSize {
		return nil, errors.New("requested segment size too large")
	}

	offset := state.AllocateEmptyStorageOffset()

	state.backingStorage.Set(*offset, IntToHash(int64(size)))

	return &SizedArbosStorageSegment{
		*offset,
		size,
		state,
	}, nil
}

func (state *ArbosState) OpenSizedSegment(offset common.Hash) *SizedArbosStorageSegment {
	rawSize := state.backingStorage.Get(offset)
	bigSize := rawSize.Big()
	if !bigSize.IsUint64() {
		panic("not a valid state segment")
	}
	size := bigSize.Uint64()
	if size == 0 {
		panic("state segment invalid or was deleted")
	} else if size > MaxSizedSegmentSize {
		panic("state segment size invalid")
	}
	return &SizedArbosStorageSegment{
		offset,
		size,
		state,
	}
}

func (seg *SizedArbosStorageSegment) Get(offset uint64) common.Hash {
	if offset >= seg.size {
		panic("out of bounds access to storage segment")
	}
	return seg.storage.backingStorage.Get(hashPlusInt(seg.offset, int64(1+offset)))
}

func (seg *SizedArbosStorageSegment) GetAsInt64(offset uint64) int64 {
	raw := seg.Get(offset).Big()
	if ! raw.IsInt64() {
		panic("out of range")
	}
	return raw.Int64()
}

func (seg *SizedArbosStorageSegment) GetAsUint64(offset uint64) uint64 {
	raw := seg.Get(offset).Big()
	if ! raw.IsUint64() {
		panic("out of range")
	}
	return raw.Uint64()
}

func (seg *SizedArbosStorageSegment) Set(offset uint64, value common.Hash) {
	if offset >= seg.size {
		panic("offset too large in SizedArbosStorageSegment::Set")
	}
	seg.storage.backingStorage.Set(hashPlusInt(seg.offset, int64(offset+1)), value)
}

func (state *ArbosState) AllocateSizedSegmentForBytes(buf []byte) *SizedArbosStorageSegment {
	sizeWords := (len(buf)+31) / 32
	seg, err := state.AllocateSizedSegment(uint64(1+sizeWords))
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

func (seg *SizedArbosStorageSegment) GetBytes() []byte {
	rawSize := seg.Get(0)

	if ! rawSize.Big().IsUint64() {
		panic("invalid segment size")
	}
	size := rawSize.Big().Uint64()
	sizeWords := (size+31) / 32
	buf := make([]byte, 32*sizeWords)
	for i := uint64(0); i < sizeWords; i++ {
		iterBuf := seg.Get(i+1).Bytes()
		for j, b := range iterBuf {
			buf[32*i+uint64(j)] = b
		}
	}
	return buf[:size]
}

func (seg *SizedArbosStorageSegment) Equals(other *SizedArbosStorageSegment) bool {
	return seg.offset == other.offset
}


