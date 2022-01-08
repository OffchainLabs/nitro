//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package addressTable

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type AddressTable struct {
	backingStorage *storage.Storage
	byAddress      *storage.Storage // 0 means item isn't in the table; n > 0 means it's in the table at slot n-1
	numItems       storage.StorageBackedUint64
}

func Initialize(sto *storage.Storage) {
}

func Open(sto *storage.Storage) *AddressTable {
	numItems := sto.OpenStorageBackedUint64(0)
	return &AddressTable{sto, sto.OpenSubStorage([]byte{}), numItems}
}

func (atab *AddressTable) Register(addr common.Address) uint64 {
	addrAsHash := common.BytesToHash(addr.Bytes())
	rev := atab.byAddress.Get(addrAsHash)
	if rev == (common.Hash{}) {
		// addr isn't in the table, so add it
		newNumItems := atab.numItems.Increment()
		atab.backingStorage.SetByUint64(newNumItems, addrAsHash)
		atab.byAddress.Set(addrAsHash, util.UintToHash(newNumItems))
		return newNumItems - 1
	} else {
		return rev.Big().Uint64() - 1
	}
}

func (atab *AddressTable) Lookup(addr common.Address) (uint64, bool) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	res := atab.byAddress.GetUint64(addrAsHash)
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
	return atab.numItems.Get()
}

func (atab *AddressTable) LookupIndex(index uint64) (common.Address, bool) {
	if index >= atab.numItems.Get() {
		return common.Address{}, false
	}
	return common.BytesToAddress(atab.backingStorage.GetByUint64(index + 1).Bytes()), true
}

func (atab *AddressTable) Compress(addr common.Address) []byte {
	index, exists := atab.Lookup(addr)
	if exists {
		return rlp.AppendUint64([]byte{}, index)
	} else {
		buf, err := rlp.EncodeToBytes(addr.Bytes())
		if err != nil {
			panic(err)
		}
		return buf
	}
}

func (atab *AddressTable) Decompress(buf []byte) (common.Address, uint64, error) {
	rd := bytes.NewReader(buf)
	decoder := rlp.NewStream(rd, 21)
	input, err := decoder.Bytes()
	if err != nil {
		return common.Address{}, 0, err
	}
	if len(input) == 20 {
		numBytesRead := uint64(rd.Size() - int64(rd.Len()))
		return common.BytesToAddress(input), numBytesRead, nil
	} else {
		rd = bytes.NewReader(buf)
		index, err := rlp.NewStream(rd, 9).Uint()
		if err != nil {
			return common.Address{}, 0, err
		}
		addr, exists := atab.LookupIndex(index)
		if !exists {
			return common.Address{}, 0, errors.New("invalid index in compressed address")
		}
		numBytesRead := uint64(rd.Size() - int64(rd.Len()))
		return addr, numBytesRead, nil
	}
}
