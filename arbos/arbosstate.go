package arbos

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)


// We use an interface since *state.stateObject is private
type GethStateObject interface {
	GetState(db state.Database, key common.Hash) common.Hash
	SetState(db state.Database, key common.Hash, value common.Hash)
}

type GethEvmStorage struct {
	state GethStateObject
	db    state.Database
}

// Use a Geth database to create an evm key-value store
func NewGethEvmStorage(statedb *state.StateDB) *GethEvmStorage {
	state := statedb.GetOrNewStateObject(common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"))
	return &GethEvmStorage{
		state: state,
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

func (store *GethEvmStorage) GetAsInt64(key common.Hash) (int64, error) {
	rawValue := store.Get(key).Big()
	if rawValue.IsInt64() {
		return rawValue.Int64(), nil
	} else {
		return 0, errors.New("expected int64 in backing storage")
	}
}

func (store *GethEvmStorage) Set(key common.Hash, value common.Hash) {
	store.state.SetState(store.db, key, value)
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
	backingStorage    *GethEvmStorage
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

func tryStorageUpgrade(backingStorage *GethEvmStorage) bool {
	formatVersion := backingStorage.Get(IntToHash(0))
	if formatVersion == IntToHash(0) {
		// we're in version 0, which is the uninitialized state; upgrade to version 1 (initialized)
		backingStorage.Set(IntToHash(0), IntToHash(1))
		backingStorage.Set(IntToHash(1), IntToHash(1024))
		backingStorage.Set(IntToHash(2), IntToHash(10000000*10*60))
		backingStorage.Set(IntToHash(3), IntToHash(10000000*60))
		backingStorage.Set(IntToHash(4), IntToHash(1000000000)) // 1 gwei
		backingStorage.Set(IntToHash(5), IntToHash(0))
		return true
	} else {
		return false
	}
}

func (state *ArbosState) FormatVersion() *big.Int {
	if state.formatVersion == nil {
		state.formatVersion = state.backingStorage.Get(IntToHash(0)).Big()
	}
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val *big.Int) {
	state.formatVersion = val
	state.backingStorage.Set(IntToHash(0), common.BigToHash(state.formatVersion))
}

func (state *ArbosState) NextAlloc() *common.Hash {
	if state.nextAlloc == nil {
		val := state.backingStorage.Get(IntToHash(1))
		state.nextAlloc = &val
	}
	return state.nextAlloc
}

func (state *ArbosState) SetNextAlloc(val *common.Hash) {
	state.nextAlloc = val
	state.backingStorage.Set(IntToHash(1), *state.nextAlloc)
}

func (state *ArbosState) GasPool() int64 {
	if state.gasPool == nil {
		val, err := state.backingStorage.GetAsInt64(IntToHash(2))
		if err != nil {
			val = 0
		}
		state.gasPool = &val
	}
	return *state.gasPool
}

func (state *ArbosState) SetGasPool(val int64) {   //BUGBUG: handle negative values correctly in storage read/write
	c := val
	state.gasPool = &c
	state.backingStorage.Set(IntToHash(2), IntToHash(c))
}

func (state *ArbosState) SmallGasPool() int64 {
	if state.smallGasPool == nil {
		val, err := state.backingStorage.GetAsInt64(IntToHash(3))
		if err != nil {
			val = 0
		}
		state.smallGasPool = &val
	}
	return *state.smallGasPool
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	c := val
	state.smallGasPool = &c
	state.backingStorage.Set(IntToHash(3), IntToHash(c))
}

func (state *ArbosState) GasPriceWei() *big.Int {
	if state.gasPriceWei == nil {
		state.gasPriceWei = state.backingStorage.Get(IntToHash(4)).Big()
	}
	return state.gasPriceWei
}

func (state *ArbosState) SetGasPriceWei(val *big.Int) {
	state.gasPriceWei = val
	state.backingStorage.Set(IntToHash(4), common.BigToHash(val))
}

func (state *ArbosState) LastTimestampSeen() *big.Int {
	if state.lastTimestampSeen == nil {
		state.lastTimestampSeen = state.backingStorage.Get(IntToHash(5)).Big()
	}
	return state.lastTimestampSeen
}

func (state *ArbosState) SetLastTimestampSeen(val *big.Int) {
	state.lastTimestampSeen = val
	state.backingStorage.Set(IntToHash(5), common.BigToHash(val))
}


type ArbosStorageSegment struct {
	offset      common.Hash
	size        uint64
	storage     *ArbosState
}

const MaxSegmentSize = 1<<48

func (state *ArbosState) AllocateSegment(size uint64) (*ArbosStorageSegment, error) {
	if size > MaxSegmentSize {
		return nil, errors.New("requested segment size too large")
	}

	offset := state.NextAlloc()
	newVal := hashPlusInt(*offset, int64(size+1))
	state.SetNextAlloc(&newVal)

	state.backingStorage.Set(*offset, IntToHash(int64(size)))

	return &ArbosStorageSegment{
		*offset,
		size,
		state,
	}, nil
}

func (state *ArbosState) OpenSegment(offset common.Hash) (*ArbosStorageSegment, error) {
	rawSize := state.backingStorage.Get(offset)
	bigSize := rawSize.Big()
	if !bigSize.IsUint64() {
		return nil, errors.New("not a valid state segment")
	}
	size := bigSize.Uint64()
	if size == 0 {
		return nil, errors.New("state segment invalid or was deleted")
	} else if size > MaxSegmentSize {
		return nil, errors.New("state segment size invalid")
	}
	return &ArbosStorageSegment{
		offset,
		size,
		state,
	}, nil
}

func (seg *ArbosStorageSegment) Get(offset uint64) (common.Hash, error) {
	if offset >= seg.size {
		return common.Hash{}, errors.New("out of bounds access to storage segment")
	}
	return seg.storage.backingStorage.Get(hashPlusInt(seg.offset, int64(1+offset))), nil
}

func (seg *ArbosStorageSegment) GetAsInt64(offset uint64) (int64, error) {
	raw, err := seg.Get(offset)
	if err != nil {
		return 0, err
	}
	rawBig := raw.Big()
	if rawBig.IsInt64() {
		return rawBig.Int64(), nil
	} else {
		return 0, errors.New("out of range")
	}
}

func (seg *ArbosStorageSegment) GetAsUint64(offset uint64) (uint64, error) {
	raw, err := seg.Get(offset)
	if err != nil {
		return 0, err
	}
	rawBig := raw.Big()
	if rawBig.IsUint64() {
		return rawBig.Uint64(), nil
	} else {
		return 0, errors.New("out of range")
	}
}

func (seg *ArbosStorageSegment) Set(offset uint64, value common.Hash) error {
	if offset >= seg.size {
		errors.New("offset too large in ArbosStorageSegment::Set")
	}
	seg.storage.backingStorage.Set(hashPlusInt(seg.offset, int64(offset+1)), value)
	return nil
}

func (state *ArbosState) AllocateSegmentForBytes(buf []byte) (*ArbosStorageSegment, error) {
	sizeWords := (len(buf)+31) / 32
	seg, err := state.AllocateSegment(uint64(1+sizeWords))
	if err != nil {
		return nil, err
	}
	if err := seg.Set(0, IntToHash(int64(len(buf)))); err != nil {
		return nil, err
	}

	offset := uint64(1)
	for len(buf) >= 32 {
		if err := seg.Set(offset, common.BytesToHash(buf[:32])); err != nil {
			return nil, err
		}
		offset += 1
		buf = buf[32:]
	}

	endBuf := [32]byte{}
	for i := 0; i < len(buf); i++ {
		endBuf[i] = buf[i]
	}
	err = seg.Set(offset, common.BytesToHash(endBuf[:]))
	if err == nil {
		return seg, nil
	} else {
		return nil, err
	}
}

func (state *ArbosState) AdvanceTimestampToAtLeast(newTimestamp *big.Int) {
	currentTimestamp := state.LastTimestampSeen()
	if newTimestamp.Cmp(currentTimestamp) > 0 {
		state.SetLastTimestampSeen(newTimestamp)
	}
}

func (seg *ArbosStorageSegment) GetBytes() ([]byte, error) {
	rawSize, err := seg.Get(0)
	if err != nil {
		return nil, err
	}

	if ! rawSize.Big().IsUint64() {
		return nil, errors.New("invalid segment size")
	}
	size := rawSize.Big().Uint64()
	sizeWords := (size+31) / 32
	buf := make([]byte, 32*sizeWords)
	for i := uint64(0); i < sizeWords; i++ {
		x, err := seg.Get(i+1)
		if err != nil {
			return nil, err
		}
		iterBuf := x.Bytes()
		for j, b := range iterBuf {
			buf[32*i+uint64(j)] = b
		}
	}
	return buf[:size], nil
}

func (seg *ArbosStorageSegment) Equals(other *ArbosStorageSegment) bool {
	return seg.offset == other.offset
}


