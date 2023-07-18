// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package addressTable

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

type AddressTable struct {
	backingStorage *storage.Storage
	byAddress      *storage.Storage // 0 means item isn't in the table; n > 0 means it's in the table at slot n-1
	numItems       storage.StorageBackedUint64
}

func Initialize(sto *storage.Storage) {
	// No initialization needed.
}

func Open(sto *storage.Storage) *AddressTable {
	numItems := sto.OpenStorageBackedUint64(0)
	return &AddressTable{sto.NoCacheCopy(), sto.OpenSubStorage([]byte{}, false), numItems}
}

func (atab *AddressTable) Register(addr common.Address) (uint64, error) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	rev, err := atab.byAddress.Get(addrAsHash)
	if err != nil {
		return 0, err
	}

	if rev != (common.Hash{}) {
		return rev.Big().Uint64() - 1, nil
	}
	// Addr isn't in the table, so add it.
	newNumItems, err := atab.numItems.Increment()
	if err != nil {
		return 0, err
	}
	if err := atab.backingStorage.SetByUint64(newNumItems, addrAsHash); err != nil {
		return 0, err
	}
	if err := atab.byAddress.Set(addrAsHash, util.UintToHash(newNumItems)); err != nil {
		return 0, err
	}
	return newNumItems - 1, nil
}

func (atab *AddressTable) Lookup(addr common.Address) (uint64, bool, error) {
	addrAsHash := common.BytesToHash(addr.Bytes())
	res, err := atab.byAddress.GetUint64(addrAsHash)
	if res == 0 || err != nil {
		return 0, false, err
	} else {
		return res - 1, true, nil
	}
}

func (atab *AddressTable) AddressExists(addr common.Address) (bool, error) {
	_, ret, err := atab.Lookup(addr)
	return ret, err
}

func (atab *AddressTable) Size() (uint64, error) {
	return atab.numItems.Get()
}

func (atab *AddressTable) LookupIndex(index uint64) (common.Address, bool, error) {
	items, err := atab.numItems.Get()
	if index >= items || err != nil {
		return common.Address{}, false, err
	}
	value, err := atab.backingStorage.GetByUint64(index + 1)
	return common.BytesToAddress(value.Bytes()), true, err
}

func (atab *AddressTable) Compress(addr common.Address) ([]byte, error) {
	index, exists, err := atab.Lookup(addr)
	if exists || err != nil {
		return rlp.AppendUint64([]byte{}, index), err
	} else {
		buf, err := rlp.EncodeToBytes(addr.Bytes())
		if err != nil {
			panic(err)
		}
		return buf, nil
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
		index, err := rlp.NewStream(rd, 9).Uint64()
		if err != nil {
			return common.Address{}, 0, err
		}
		addr, exists, err := atab.LookupIndex(index)
		if err != nil {
			return common.Address{}, 0, err
		}
		if !exists {
			return common.Address{}, 0, errors.New("invalid index in compressed address")
		}
		numBytesRead := uint64(rd.Size() - int64(rd.Len()))
		return addr, numBytesRead, nil
	}
}
