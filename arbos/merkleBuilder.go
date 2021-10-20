//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"io"
)

type MerkleBuilder struct {
	size     uint64
	partials [][]byte
}

func NewBuilder() *MerkleBuilder {
	return &MerkleBuilder{0, make([][]byte, 0)}
}

func (b *MerkleBuilder) Append(itemHash common.Hash) {
	b.size++
	level := 0
	soFar := itemHash.Bytes()
	for {
		if level == len(b.partials) {
			b.partials = append(b.partials, soFar)
			return
		}
		if b.partials[level] == nil {
			b.partials[level] = soFar
			return
		}
		soFar = crypto.Keccak256(b.partials[level], soFar)
		b.partials[level] = nil
		level += 1
	}
}

func (b *MerkleBuilder) Size() uint64 {
	return b.size
}

func (b *MerkleBuilder) Root() common.Hash {
	if b.size == 0 {
		return common.Hash{}
	}
	if b.size == 1 {
		return common.BytesToHash(b.partials[0])
	}
	ret := make([]byte, 32)
	emptySoFar := true
	for i := 0; i < len(b.partials); i++ {
		if b.partials[i] == nil {
			if !emptySoFar {
				ret = crypto.Keccak256(make([]byte, 32), ret)
			}
		} else {
			if emptySoFar {
				if i+1 == len(b.partials) {
					ret = b.partials[i]
				} else {
					emptySoFar = false
					ret = crypto.Keccak256(b.partials[i], make([]byte, 32))
				}
			} else {
				ret = crypto.Keccak256(b.partials[i], ret)
			}
		}
	}

	return common.BytesToHash(ret)
}

func (b *MerkleBuilder) Serialize(wr io.Writer) error {
	if err := Uint64ToWriter(b.size, wr); err != nil {
		return err
	}
	if err := Uint64ToWriter(uint64(len(b.partials)), wr); err != nil {
		return err
	}
	for _, partial := range b.partials {
		if partial == nil {
			buf := []byte{0}
			if _, err := wr.Write(buf); err != nil {
				return err
			}
		} else {
			buf := []byte{1}
			if _, err := wr.Write(buf); err != nil {
				return err
			}
			if _, err := wr.Write(partial); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewBuilderFromReader(rd io.Reader) (*MerkleBuilder, error) {
	size, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	numPartials, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	partials := make([][]byte, numPartials)
	for i := range partials {
		var buf [1]byte
		if _, err := rd.Read(buf[:]); err != nil {
			return nil, err
		}
		if buf[0] != 0 {
			buf32 := make([]byte, 32)
			if _, err := io.ReadFull(rd, buf32); err != nil {
				return nil, err
			}
			partials[i] = buf32[:]
		}
	}
	return &MerkleBuilder{size, partials}, nil
}

func (b *MerkleBuilder) Persist(segment *StorageSegment) {
	segment.Set(0, IntToHash(int64(b.size)))
	segment.Set(1, IntToHash(int64(len(b.partials))))
	for i, partial := range b.partials {
		segment.Set(uint64(i+2), common.BytesToHash(partial))
	}
}

func NewBuilderFromSegment(segment *StorageSegment) *MerkleBuilder {
	size := segment.Get(0).Big().Uint64()
	numPartials := segment.Get(1).Big().Uint64()
	partials := make([][]byte, numPartials)
	for i, _ := range partials {
		partials[i] = segment.Get(uint64(i + 2)).Bytes()
	}
	return &MerkleBuilder{ size, partials }
}
