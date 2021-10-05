package arbos

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"math/big"
)

type BackingEvmStorage interface {
	Get(offset common.Hash) common.Hash
	GetAsInt64(offset common.Hash) (int64, error)
	Set(offset common.Hash, value common.Hash)
}

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
	formatVersion     common.Hash
	nextAlloc         common.Hash
	gasPool           int64
	smallGasPool      int64
	gasPriceWei       common.Hash
	lastTimestampSeen common.Hash
	backingStorage    BackingEvmStorage
	statedb           *state.StateDB
}

func OpenArbosState(statedb *state.StateDB, timestamp common.Hash) *ArbosState {
	
	backingStorage := NewGethEvmStorage(statedb)
	
	for tryStorageUpgrade(backingStorage, timestamp) {}
	
	formatVersion := backingStorage.Get(IntToHash(0))
	nextAlloc := backingStorage.Get(IntToHash(1))
	gasPool, err := backingStorage.GetAsInt64(IntToHash(2))
	if err != nil {
		panic(err)
	}
	smallGasPool, err := backingStorage.GetAsInt64(IntToHash(3))
	if err != nil {
		panic(err)
	}
	gasPriceWei := backingStorage.Get(IntToHash(4))
	lastTimestampSeen := backingStorage.Get(IntToHash(5))
	return &ArbosState{
		formatVersion,
		nextAlloc,
		gasPool,
		smallGasPool,
		gasPriceWei,
		lastTimestampSeen,
		backingStorage,
		statedb,
	}
}

func tryStorageUpgrade(backingStorage BackingEvmStorage, timestamp common.Hash) bool {
	formatVersion := backingStorage.Get(IntToHash(0))
	if formatVersion == IntToHash(0) {
		// we're in version 0, which is the uninitialized state; upgrade to version 1 (initialized)
		backingStorage.Set(IntToHash(0), IntToHash(1))
		backingStorage.Set(IntToHash(1), IntToHash(1024))
		backingStorage.Set(IntToHash(2), IntToHash(10000000*10*60))
		backingStorage.Set(IntToHash(3), IntToHash(10000000*60))
		backingStorage.Set(IntToHash(4), IntToHash(1000000000)) // 1 gwei
		backingStorage.Set(IntToHash(5), timestamp)
		return true
	} else {
		return false
	}
}

func (state *ArbosState) FormatVersion() common.Hash {
	return state.formatVersion
}

func (state *ArbosState) SetFormatVersion(val common.Hash) {
	state.formatVersion = val
	state.backingStorage.Set(IntToHash(0), state.formatVersion)
}

func (state *ArbosState) NextAlloc() common.Hash {
	return state.nextAlloc
}

func (state *ArbosState) SetNextAlloc(val common.Hash) {
	state.nextAlloc = val
	state.backingStorage.Set(IntToHash(1), state.nextAlloc)
}

func (state *ArbosState) GasPool() int64 {
	return state.gasPool
}

func (state *ArbosState) SetGasPool(val int64) {
	state.gasPool = val
	state.backingStorage.Set(IntToHash(2), IntToHash(val))
}

func (state *ArbosState) SmallGasPool() int64 {
	return state.smallGasPool
}

func (state *ArbosState) SetSmallGasPool(val int64) {
	state.smallGasPool = val
	state.backingStorage.Set(IntToHash(3), IntToHash(val))
}

func (state *ArbosState) GasPriceWei() common.Hash {
	return state.gasPriceWei
}

func (state *ArbosState) SetGasPriceWei(val common.Hash) {
	state.gasPriceWei = val
	state.backingStorage.Set(IntToHash(4), val)
}

func (state *ArbosState) LastTimestampSeen() common.Hash {
	return state.lastTimestampSeen
}

func (state *ArbosState) SetLastTimestampSeen(val common.Hash) {
	state.lastTimestampSeen = val
	state.backingStorage.Set(IntToHash(5), val)
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
	state.SetNextAlloc(hashPlusInt(state.nextAlloc, int64(size+1)))

	state.backingStorage.Set(offset, IntToHash(int64(size)))

	return &ArbosStorageSegment{
		offset,
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

func (state *ArbosState) AdvanceTimestampToAtLeast(timestamp common.Hash) {
	newTimestampBig := timestamp.Big()
	currentTimestampBig := state.LastTimestampSeen().Big()
	if newTimestampBig.Cmp(currentTimestampBig) > 0 {
		state.SetLastTimestampSeen(timestamp)
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
