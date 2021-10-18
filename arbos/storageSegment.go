package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type StorageSegment struct {
	offset      common.Hash
	size        uint64
	storage     EvmStorage
}

const MaxSizedSegmentSize = 1<<48

func (seg *StorageSegment) Get(offset uint64) common.Hash {
	if offset >= seg.size {
		panic("out of bounds access to storage segment")
	}
	return seg.storage.Get(hashPlusInt(seg.offset, int64(1+offset)))
}

func (seg *StorageSegment) GetAsInt64(offset uint64) int64 {
	raw := seg.Get(offset).Big()
	if ! raw.IsInt64() {
		panic("out of range")
	}
	return raw.Int64()
}

func (seg *StorageSegment) GetAsUint64(offset uint64) uint64 {
	raw := seg.Get(offset).Big()
	if ! raw.IsUint64() {
		panic("out of range")
	}
	return raw.Uint64()
}

func (seg *StorageSegment) Set(offset uint64, value common.Hash) {
	if offset >= seg.size {
		panic("offset too large in StorageSegment::Set")
	}
	seg.storage.Set(hashPlusInt(seg.offset, int64(offset+1)), value)
}

func (seg *StorageSegment) GetBytes() []byte {
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

func (seg *StorageSegment) WriteBytes(buf []byte) {
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
}

func (seg *StorageSegment) Delete() {
	seg.storage.Set(seg.offset, common.Hash{})
	for i := uint64(0); i < seg.size; i++ {
		offset := new(big.Int).Add(seg.offset.Big(), big.NewInt(int64(i)))
		seg.storage.Set(common.BigToHash(offset), common.Hash{})
	}
}

func (seg *StorageSegment) Equals(other *StorageSegment) bool {
	return seg.offset == other.offset
}
