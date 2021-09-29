package arbos

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type ArbosStorage struct {
	formatVersion  common.Hash
	nextAlloc      common.Hash
	backingStorage BackingEvmStorage
}

type BackingEvmStorage interface {
	Get(offset common.Hash) common.Hash
	Set(offset common.Hash, value common.Hash)
}

type MemoryBackingEvmStorage struct {
	contents map[common.Hash]common.Hash
}

func NewMemoryBackingEvmStorage() *MemoryBackingEvmStorage {
	return &MemoryBackingEvmStorage{
		make(map[common.Hash]common.Hash),
	}
}

func (st *MemoryBackingEvmStorage) Get(offset common.Hash) common.Hash {
	value, exists := st.contents[offset]
	if exists {
		return value
	} else {
		return common.Hash{}   // empty slot is treated as zero
	}
}

func (st *MemoryBackingEvmStorage) Set(offset common.Hash, value common.Hash) {
	st.contents[offset] = value
}

func intToHash(val int64) common.Hash {
	return common.BigToHash(big.NewInt(val))
}

func hashPlusInt(x common.Hash, y int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(x.Big(), big.NewInt(y)))
}

func OpenArbosStorage(backingStorage BackingEvmStorage) *ArbosStorage {
	formatVersion := backingStorage.Get(intToHash(0))
	nextAlloc := backingStorage.Get(intToHash(1))
	storage := &ArbosStorage{
		formatVersion,
		nextAlloc,
		backingStorage,
	}

	for storage.tryUpgrade() {}

	return storage
}

func (storage *ArbosStorage) tryUpgrade() bool {
	if storage.formatVersion == intToHash(0) {
		// we're in version 0, which is the uninitialized state; upgrade to version 1 (initialized)
		storage.setNextAlloc(intToHash(1024));
		storage.setFormatVersion(intToHash(1));
		return true
	} else {
		return false
	}
}

func (storage*ArbosStorage) setFormatVersion(val common.Hash) {
	storage.formatVersion = val
	storage.backingStorage.Set(intToHash(0), storage.formatVersion)
}

func (storage*ArbosStorage) setNextAlloc(val common.Hash) {
	storage.nextAlloc = val
	storage.backingStorage.Set(intToHash(1), storage.nextAlloc)
}

type ArbosStorageSegment struct {
	offset      common.Hash
	size        uint64
	storage     *ArbosStorage
}

const MaxSegmentSize = 1<<48

func (storage*ArbosStorage) Allocate(size uint64) (*ArbosStorageSegment, error) {
	if size > MaxSegmentSize {
		return nil, errors.New("requested segment size too large")
	}

	offset := storage.nextAlloc
	storage.nextAlloc = hashPlusInt(storage.nextAlloc, int64(size+1))
	storage.backingStorage.Set(intToHash(1), storage.nextAlloc)

	storage.backingStorage.Set(offset, intToHash(int64(size)))

	return &ArbosStorageSegment{
		offset,
		size,
		storage,
	}, nil
}

func (storage *ArbosStorage) Open(offset common.Hash) (*ArbosStorageSegment, error) {
	rawSize := storage.backingStorage.Get(offset)
	bigSize := rawSize.Big()
	if !bigSize.IsUint64() {
		return nil, errors.New("not a valid storage segment")
	}
	size := bigSize.Uint64()
	if size == 0 {
		return nil, errors.New("storage segment invalid or was deleted")
	} else if size > MaxSegmentSize {
		return nil, errors.New("storage segment size invalid")
	}
	return &ArbosStorageSegment{
		offset,
		size,
		storage,
	}, nil
}

func (seg *ArbosStorageSegment) Get(offset uint64) (common.Hash, error) {
	if offset >= seg.size {
		return common.Hash{}, errors.New("out of bounds access to storage segment")
	}
	return seg.storage.backingStorage.Get(hashPlusInt(seg.offset, int64(1+offset))), nil
}

func (seg *ArbosStorageSegment) Set(offset uint64, value common.Hash) error {
	if offset >= seg.size {
		errors.New("offset too large in ArbosStorageSegment::Set")
	}
	seg.storage.backingStorage.Set(hashPlusInt(seg.offset, int64(offset+1)), value)
	return nil
}

func (storage*ArbosStorage) AllocateForBytes(buf []byte) (*ArbosStorageSegment, error) {
	sizeWords := (len(buf)+31) / 32
	seg, err := storage.Allocate(uint64(1+sizeWords))
	if err != nil {
		return nil, err
	}
	if err := seg.Set(0, intToHash(int64(len(buf)))); err != nil {
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


