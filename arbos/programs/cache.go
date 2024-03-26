// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

/*type initTable struct {
	backingStorage *storage.Storage
	bits           storage.StorageBackedUint8
	base           common.Hash
}

const initTableBitsOffset uint64 = iota

func initInitTable(sto *storage.Storage) {
	bits := sto.OpenStorageBackedUint8(initTableBitsOffset)
	_ = bits.Set(initialInitTableBits)
}

func openInitTable(sto *storage.Storage) *initTable {
	return &initTable{
		backingStorage: sto,
		bits:           sto.OpenStorageBackedUint8(initTableBitsOffset),
		base:           sto.AbsoluteKey(common.Hash{}),
	}
}

func (table initTable) insert(moduleHash common.Hash, db *state.StateDB) bool {
	bits, err := table.bits.Get()
	table.backingStorage.Burner().Restrict(err)

	size := uint32(1) << bits
	index := util.Uint32ToHash(arbmath.BytesToUint32(moduleHash[:4]) % size)
	slot := table.backingStorage.AbsoluteKey(index)

	stored := db.GetState(types.ArbosStateAddress, slot)
	if stored == moduleHash {
		return true
	}
	db.SetState(types.ArbosStateAddress, slot, moduleHash)
	return false
}

type trieTable struct {
	backingStorage *storage.Storage
	bits           storage.StorageBackedUint8
}

const trieTableBitsOffset uint64 = iota

func initTrieTable(sto *storage.Storage) {
	bits := sto.OpenStorageBackedUint8(trieTableBitsOffset)
	_ = bits.Set(initialTrieTableBits)
}

func openTrieTable(sto *storage.Storage) *trieTable {
	return &trieTable{
		backingStorage: sto,
		bits:           sto.OpenStorageBackedUint8(trieTableBitsOffset),
	}
        }*/

/*func (table trieTable) contains(addy common.Address, key common.Hash, db *state.StateDB) bool {
	slot := table.offset(addy, key)
	stored := db.GetState(types.ArbosStateAddress, slot)
	return stored == item
}

func (table trieTable) insert(item common.Hash, db *state.StateDB) {
	slot := table.offset(item)
	db.SetState(types.ArbosStateAddress, slot, item)
}

func (table trieTable) offset(addy common.Hash) common.Hash {
	bits, err := table.bits.Get()
	table.backingStorage.Burner().Restrict(err)

	size := uint32(1) << bits
	index := util.Uint32ToHash(arbmath.BytesToUint32(item[:4]) % size / 2 * 2)
	return table.backingStorage.AbsoluteKey(index)
        }*/
