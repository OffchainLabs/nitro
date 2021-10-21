//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressTable

import (
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type AddressTable struct {
	backingStorage *storage.Storage
	byAddress      *storage.Storage // 0 means item isn't in the table; n > 0 means it's in the table at slot n-1
	numItems       uint64
}

func Initialize(sto *storage.Storage) {
}

func Open(sto *storage.Storage) *AddressTable {
	numItems := sto.GetByInt64(0).Big().Uint64()
	return &AddressTable{sto, sto.OpenSubStorage([]byte{}), numItems}
}

func (atab *AddressTable) Register(addr common.Address) uint64 {
	addrAsHash := common.BytesToHash(addr.Bytes())
	rev := atab.byAddress.Get(addrAsHash)
	if rev == (common.Hash{}) {
		// addr isn't in the table, so add it
		ret := atab.numItems
		atab.numItems++
		atab.backingStorage.SetByInt64(0, util.IntToHash(int64(atab.numItems)))
		atab.backingStorage.SetByInt64(int64(ret+1), addrAsHash)
		atab.byAddress.Set(addrAsHash, util.IntToHash(int64(ret+1)))
		return ret
	} else {
		return rev.Big().Uint64() - 1
	}
}

func (atab *AddressTable) Lookup(addr common.Address) (uint64, bool) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	res := atab.byAddress.Get(addrAsHash).Big().Uint64()
	if res == 0 {
		return 0, false
	} else {
		return res - 1, true
	}
}

func (atab *AddressTable) AddressExists(addr common.Address) bool {
	_, ret := atab.Lookup(addr)
	return ret
}

func (atab *AddressTable) Size() uint64 {
	return atab.numItems
}

func (atab *AddressTable) LookupIndex(index uint64) (common.Address, bool) {
	if index >= atab.numItems {
		return common.Address{}, false
	}
	return common.BytesToAddress(atab.backingStorage.GetByInt64(int64(index + 1)).Bytes()), true
}

// In compression and decompression, we use a vastly simplified (but compatible) implementation of RLP encode/decode.
// This saves us from having to bring in a whole RLP library when our needs are very simple here.

const RLPPrefixFor8Bytes byte = 128 + 8
const RLPPrefixFor20Bytes byte = 128 + 20

func (atab *AddressTable) Compress(addr common.Address) []byte {
	index, exists := atab.Lookup(addr)
	if exists {
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], index)
		return append([]byte{RLPPrefixFor8Bytes}, buf[:]...)
	} else {
		return append([]byte{RLPPrefixFor20Bytes}, addr.Bytes()...)
	}
}

func (atab *AddressTable) Decompress(buf []byte) (common.Address, uint64, error) {
	switch buf[0] {
	case RLPPrefixFor8Bytes:
		index := binary.BigEndian.Uint64(buf[1:9])
		addr, exists := atab.LookupIndex(index)
		if !exists {
			return common.Address{}, 0, errors.New("invalid compressed address")
		}
		return addr, 9, nil
	case RLPPrefixFor20Bytes:
		return common.BytesToAddress(buf[1:21]), 21, nil
	default:
		return common.Address{}, 0, errors.New("invalid compressed address format")
	}
}
