//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package segment

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type Segment struct {
	Offset  common.Hash
	Size    uint64
	storage storage.Storage
}

const MaxSizedSegmentSize = 1 << 48

func New(offset common.Hash, size uint64, storage storage.Storage) *Segment {
	return &Segment{
		offset,
		size,
		storage,
	}
}

func (seg *Segment) Get(offset uint64) common.Hash {
	if offset >= seg.Size {
		panic("out of bounds access to storage segment")
	}
	return seg.storage.Get(util.HashPlusInt(seg.Offset, int64(1+offset)))
}

func (seg *Segment) GetAsInt64(offset uint64) int64 {
	raw := seg.Get(offset).Big()
	if !raw.IsInt64() {
		panic("out of range")
	}
	return raw.Int64()
}

func (seg *Segment) GetAsUint64(offset uint64) uint64 {
	raw := seg.Get(offset).Big()
	if !raw.IsUint64() {
		panic("out of range")
	}
	return raw.Uint64()
}

func (seg *Segment) Set(offset uint64, value common.Hash) {
	if offset >= seg.Size {
		panic("Offset too large in storage::Set")
	}
	seg.storage.Set(util.HashPlusInt(seg.Offset, int64(offset+1)), value)
}

func (seg *Segment) GetBytes() []byte {
	rawSize := seg.Get(0)

	if !rawSize.Big().IsUint64() {
		panic("invalid segment Size")
	}
	size := rawSize.Big().Uint64()
	sizeWords := (size + 31) / 32
	buf := make([]byte, 32*sizeWords)
	for i := uint64(0); i < sizeWords; i++ {
		iterBuf := seg.Get(i + 1).Bytes()
		for j, b := range iterBuf {
			buf[32*i+uint64(j)] = b
		}
	}
	return buf[:size]
}

func (seg *Segment) WriteBytes(buf []byte) {
	seg.Set(0, util.IntToHash(int64(len(buf))))

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

func (seg *Segment) Delete() {
	seg.storage.Set(seg.Offset, common.Hash{})
	for i := uint64(0); i < seg.Size; i++ {
		offset := new(big.Int).Add(seg.Offset.Big(), big.NewInt(int64(i)))
		seg.storage.Set(common.BigToHash(offset), common.Hash{})
	}
}

func (seg *Segment) Equals(other *Segment) bool {
	return seg.Offset == other.Offset
}
