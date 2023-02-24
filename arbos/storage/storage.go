// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package storage

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Storage allows ArbOS to store data persistently in the Ethereum-compatible stateDB. This is represented in
// the stateDB as the storage of a fictional Ethereum account at address 0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF.
//
// The storage is logically a tree of storage spaces which can be nested hierarchically, with each storage space
// containing a key-value store with 256-bit keys and values. Uninitialized storage spaces and uninitialized keys
// within initialized storage spaces are deemed to be filled with zeroes (consistent with the behavior of Ethereum
// account storage). Logically, when a chain is launched, all possible storage spaces and all possible keys within
// them exist and contain zeroes.
//
// A storage space (represented by a Storage object) has a byte-slice storageKey which distinguishes it from other
// storage spaces. The root Storage has its storageKey as the empty string. A parent storage space can contain children,
// each with a distinct name. The storageKey of a child is keccak256(parent.storageKey, name). Note that two spaces
// cannot have the same storageKey because that would imply a collision in keccak256.
//
// The contents of all storage spaces are stored in a single, flat key-value store that is implemented as the storage
// of the fictional Ethereum account. The contents of key, within a storage space with storageKey, are stored
// at location keccak256(storageKey, key) in the flat KVS. Two slots, whether in the same or different storage spaces,
// cannot occupy the same location because that would imply a collision in keccak256.

type Storage struct {
	account    common.Address
	db         vm.StateDB
	storageKey []byte
	burner     burn.Burner
}

// the following constants have been deprecated as of ArbOS v11
const StorageReadCostV0 = params.SloadGasEIP2200
const StorageWriteCostV0 = params.SstoreSetGasEIP2200
const StorageWriteZeroCostV0 = params.SstoreResetGasEIP2200

// NewGeth uses a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB, burner burn.Burner) *Storage {
	statedb.SetNonce(types.ArbosStateAddress, 1) // setting the nonce ensures Geth won't treat ArbOS as empty

	// We may have reached ArbOS without having passed through the canonical STF
	// In these cases, record the fact that we've touched ArbOS
	if !burner.OutsideTx() {
		statedb.AddAddressToAccessList(types.ArbosStateAddress)
		statedb.AddSlotToAccessList(types.ArbosStateAddress, common.Hash{})
	}
	return &Storage{
		account:    types.ArbosStateAddress,
		db:         statedb,
		storageKey: []byte{},
		burner:     burner,
	}
}

// NewMemoryBacked uses Geth's memory-backed database to create an evm key-value store
func NewMemoryBacked(burner burn.Burner) *Storage {
	return NewGeth(NewMemoryBackedStateDB(), burner)
}

// NewMemoryBackedStateDB uses Geth's memory-backed database to create a statedb
func NewMemoryBackedStateDB() vm.StateDB {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		panic("failed to init empty statedb")
	}
	return statedb
}

// We map addresses using "pages" of 256 storage slots. We hash over the page number but not the offset within
// a page, to preserve contiguity within a page. This will reduce cost if/when Ethereum switches to storage
// representations that reward contiguity.
// Because page numbers are 248 bits, this gives us 124-bit security against collision attacks, which is good enough.
func mapAddress(storageKey []byte, key common.Hash) common.Hash {
	keyBytes := key.Bytes()
	boundary := common.HashLength - 1
	return common.BytesToHash(
		append(
			crypto.Keccak256(storageKey, keyBytes[:boundary])[:boundary],
			keyBytes[boundary],
		),
	)
}

func chargeForRead(burner burn.Burner, db vm.StateDB, key common.Hash) error {
	pricingV0 := func() error {
		return burner.Burn(StorageReadCostV0)
	}
	if burner.OutsideTx() {
		return pricingV0()
	}

	// get the cost early to update access lists even when acting as an old version
	cost := vm.StateLoadCost(db, types.ArbosStateAddress, key)
	if burner.Version() < 11 {
		return pricingV0()
	}
	return burner.Burn(cost)
}

func chargeForWrite(burner burn.Burner, db vm.StateDB, key, value common.Hash) error {
	pricingV0 := func() error {
		if value == (common.Hash{}) {
			return burner.Burn(StorageWriteZeroCostV0)
		}
		return burner.Burn(StorageWriteCostV0)
	}
	if burner.OutsideTx() {
		return pricingV0()
	}

	// do StateStoreCost's sentry check before computing the state store cost
	if burner.Version() > 11 {
		if err := burner.RequireGas(params.SstoreSentryGasEIP2200); err != nil {
			return err
		}
	}

	// get the cost early to update access lists even when acting as an old version
	cost := vm.StateStoreCost(db, types.ArbosStateAddress, key, value)
	if burner.Version() < 11 {
		return pricingV0()
	}
	return burner.Burn(cost)
}

func (store *Storage) Account() common.Address {
	return store.account
}

func (store *Storage) Get(key common.Hash) (common.Hash, error) {
	err := chargeForRead(store.burner, store.db, key)
	if err != nil {
		return common.Hash{}, err
	}
	if info := store.burner.TracingInfo(); info != nil {
		info.RecordStorageGet(key)
	}
	return store.db.GetState(store.account, mapAddress(store.storageKey, key)), nil
}

func (store *Storage) GetStorageSlot(key common.Hash) common.Hash {
	return mapAddress(store.storageKey, key)
}

func (store *Storage) GetUint64(key common.Hash) (uint64, error) {
	value, err := store.Get(key)
	return value.Big().Uint64(), err
}

func (store *Storage) GetByUint64(key uint64) (common.Hash, error) {
	return store.Get(util.UintToHash(key))
}

func (store *Storage) GetUint64ByUint64(key uint64) (uint64, error) {
	return store.GetUint64(util.UintToHash(key))
}

func (store *Storage) Set(key common.Hash, value common.Hash) error {
	if store.burner.ReadOnly() {
		log.Error("Read-only burner attempted to mutate state", "key", key, "value", value)
		return vm.ErrWriteProtection
	}
	err := chargeForWrite(store.burner, store.db, key, value)
	if err != nil {
		return err
	}
	if info := store.burner.TracingInfo(); info != nil {
		info.RecordStorageSet(key, value)
	}
	store.db.SetState(store.account, mapAddress(store.storageKey, key), value)
	return nil
}

func (store *Storage) SetByUint64(key uint64, value common.Hash) error {
	return store.Set(util.UintToHash(key), value)
}

func (store *Storage) SetUint64ByUint64(key uint64, value uint64) error {
	return store.Set(util.UintToHash(key), util.UintToHash(value))
}

func (store *Storage) Clear(key common.Hash) error {
	return store.Set(key, common.Hash{})
}

func (store *Storage) ClearByUint64(key uint64) error {
	return store.Set(util.UintToHash(key), common.Hash{})
}

func (store *Storage) Swap(key common.Hash, newValue common.Hash) (common.Hash, error) {
	oldValue, err := store.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	return oldValue, store.Set(key, newValue)
}

func (store *Storage) OpenSubStorage(id []byte) *Storage {
	return &Storage{
		store.account,
		store.db,
		crypto.Keccak256(store.storageKey, id),
		store.burner,
	}
}

func (store *Storage) SetBytes(b []byte) error {
	err := store.ClearBytes()
	if err != nil {
		return err
	}
	err = store.SetUint64ByUint64(0, uint64(len(b)))
	if err != nil {
		return err
	}
	offset := uint64(1)
	for len(b) >= 32 {
		err = store.SetByUint64(offset, common.BytesToHash(b[:32]))
		if err != nil {
			return err
		}
		b = b[32:]
		offset++
	}
	return store.SetByUint64(offset, common.BytesToHash(b))
}

func (store *Storage) GetBytes() ([]byte, error) {
	bytesLeft, err := store.GetUint64ByUint64(0)
	if err != nil {
		return nil, err
	}
	ret := []byte{}
	offset := uint64(1)
	for bytesLeft >= 32 {
		next, err := store.GetByUint64(offset)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next.Bytes()...)
		bytesLeft -= 32
		offset++
	}
	next, err := store.GetByUint64(offset)
	if err != nil {
		return nil, err
	}
	ret = append(ret, next.Bytes()[32-bytesLeft:]...)
	return ret, nil
}

func (store *Storage) GetBytesSize() (uint64, error) {
	return store.GetUint64ByUint64(0)
}

func (store *Storage) ClearBytes() error {
	bytesLeft, err := store.GetUint64ByUint64(0)
	if err != nil {
		return err
	}
	offset := uint64(1)
	for bytesLeft > 0 {
		err := store.ClearByUint64(offset)
		if err != nil {
			return err
		}
		offset++
		if bytesLeft < 32 {
			bytesLeft = 0
		} else {
			bytesLeft -= 32
		}
	}
	return store.ClearByUint64(0)
}

func (store *Storage) Burner() burn.Burner {
	return store.burner // not public because these should never be changed once set
}

func (store *Storage) Keccak(data ...[]byte) ([]byte, error) {
	byteCount := 0
	for _, part := range data {
		byteCount += len(part)
	}
	cost := 30 + 6*arbmath.WordsForBytes(uint64(byteCount))
	if err := store.burner.Burn(cost); err != nil {
		return nil, err
	}
	return crypto.Keccak256(data...), nil
}

func (store *Storage) KeccakHash(data ...[]byte) (common.Hash, error) {
	bytes, err := store.Keccak(data...)
	return common.BytesToHash(bytes), err
}

type StorageSlot struct {
	account common.Address
	db      vm.StateDB
	slot    common.Hash
	burner  burn.Burner
}

func (store *Storage) NewSlot(offset uint64) StorageSlot {
	return StorageSlot{store.account, store.db, mapAddress(store.storageKey, util.UintToHash(offset)), store.burner}
}

func (ss *StorageSlot) Get() (common.Hash, error) {
	err := chargeForRead(ss.burner, ss.db, ss.slot)
	if err != nil {
		return common.Hash{}, err
	}
	if info := ss.burner.TracingInfo(); info != nil {
		info.RecordStorageGet(ss.slot)
	}
	return ss.db.GetState(ss.account, ss.slot), nil
}

func (ss *StorageSlot) Set(value common.Hash) error {
	if ss.burner.ReadOnly() {
		log.Error("Read-only burner attempted to mutate state", "value", value)
		return vm.ErrWriteProtection
	}
	err := chargeForWrite(ss.burner, ss.db, ss.slot, value)
	if err != nil {
		return err
	}
	if info := ss.burner.TracingInfo(); info != nil {
		info.RecordStorageSet(ss.slot, value)
	}
	ss.db.SetState(ss.account, ss.slot, value)
	return nil
}

// StorageBackedInt64 is an int64 stored inside the StateDB.
// Implementation note: Conversions between big.Int and common.Hash give weird results
// for negative values, so we cast to uint64 before writing to storage and cast back to int64 after reading.
// Golang casting between uint64 and int64 doesn't change the data, it just reinterprets the same 8 bytes,
// so this is a hacky but reliable way to store an 8-byte int64 in a common.Hash storage slot.
type StorageBackedInt64 struct {
	StorageSlot
}

func (store *Storage) OpenStorageBackedInt64(offset uint64) StorageBackedInt64 {
	return StorageBackedInt64{store.NewSlot(offset)}
}

func (sbu *StorageBackedInt64) Get() (int64, error) {
	raw, err := sbu.StorageSlot.Get()
	if !raw.Big().IsUint64() {
		panic("invalid value found in StorageBackedInt64 storage")
	}
	return int64(raw.Big().Uint64()), err // see implementation note above
}

func (sbu *StorageBackedInt64) Set(value int64) error {
	return sbu.StorageSlot.Set(util.UintToHash(uint64(value))) // see implementation note above
}

// StorageBackedBips represents a number of basis points
type StorageBackedBips struct {
	backing StorageBackedInt64
}

func (store *Storage) OpenStorageBackedBips(offset uint64) StorageBackedBips {
	return StorageBackedBips{StorageBackedInt64{store.NewSlot(offset)}}
}

func (sbu *StorageBackedBips) Get() (arbmath.Bips, error) {
	value, err := sbu.backing.Get()
	return arbmath.Bips(value), err
}

func (sbu *StorageBackedBips) Set(bips arbmath.Bips) error {
	return sbu.backing.Set(int64(bips))
}

type StorageBackedUint64 struct {
	StorageSlot
}

func (store *Storage) OpenStorageBackedUint64(offset uint64) StorageBackedUint64 {
	return StorageBackedUint64{store.NewSlot(offset)}
}

func (sbu *StorageBackedUint64) Get() (uint64, error) {
	raw, err := sbu.StorageSlot.Get()
	if !raw.Big().IsUint64() {
		panic("expected uint64 compatible value in storage")
	}
	return raw.Big().Uint64(), err
}

func (sbu *StorageBackedUint64) Set(value uint64) error {
	bigValue := new(big.Int).SetUint64(value)
	return sbu.StorageSlot.Set(common.BigToHash(bigValue))
}

func (sbu *StorageBackedUint64) Clear() error {
	return sbu.Set(0)
}

func (sbu *StorageBackedUint64) Increment() (uint64, error) {
	old, err := sbu.Get()
	if err != nil {
		return 0, err
	}
	if old+1 < old {
		panic("Overflow in StorageBackedUint64::Increment")
	}
	return old + 1, sbu.Set(old + 1)
}

func (sbu *StorageBackedUint64) Decrement() (uint64, error) {
	old, err := sbu.Get()
	if err != nil {
		return 0, err
	}
	if old == 0 {
		panic("Underflow in StorageBackedUint64::Decrement")
	}
	return old - 1, sbu.Set(old - 1)
}

type MemoryBackedUint64 struct {
	contents uint64
}

func (mbu *MemoryBackedUint64) Get() (uint64, error) {
	return mbu.contents, nil
}

func (mbu *MemoryBackedUint64) Set(val uint64) error {
	mbu.contents = val
	return nil
}

func (mbu *MemoryBackedUint64) Increment() (uint64, error) {
	old := mbu.contents
	if old+1 < old {
		panic("Overflow in MemoryBackedUint64::Increment")
	}
	return old + 1, mbu.Set(old + 1)
}

func (mbu *MemoryBackedUint64) Decrement() (uint64, error) {
	old := mbu.contents
	if old == 0 {
		panic("Underflow in MemoryBackedUint64::Decrement")
	}
	return old - 1, mbu.Set(old - 1)
}

type WrappedUint64 interface {
	Get() (uint64, error)
	Set(uint64) error
	Increment() (uint64, error)
	Decrement() (uint64, error)
}

var twoToThe256 = new(big.Int).Lsh(common.Big1, 256)
var twoToThe256MinusOne = new(big.Int).Sub(twoToThe256, common.Big1)
var twoToThe255 = new(big.Int).Lsh(common.Big1, 255)
var twoToThe255MinusOne = new(big.Int).Sub(twoToThe255, common.Big1)

type StorageBackedBigUint struct {
	StorageSlot
}

func (store *Storage) OpenStorageBackedBigUint(offset uint64) StorageBackedBigUint {
	return StorageBackedBigUint{store.NewSlot(offset)}
}

func (sbbu *StorageBackedBigUint) Get() (*big.Int, error) {
	asHash, err := sbbu.StorageSlot.Get()
	if err != nil {
		return nil, err
	}
	return asHash.Big(), nil
}

// Warning: this will panic if it underflows or overflows with a system burner
// SetSaturatingWithWarning is likely better
func (sbbu *StorageBackedBigUint) SetChecked(val *big.Int) error {
	if val.Sign() < 0 {
		return sbbu.burner.HandleError(fmt.Errorf("underflow in StorageBackedBigUint.Set setting value %v", val))
	} else if val.BitLen() > 256 {
		return sbbu.burner.HandleError(fmt.Errorf("overflow in StorageBackedBigUint.Set setting value %v", val))
	}
	return sbbu.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

func (sbbu *StorageBackedBigUint) SetSaturatingWithWarning(val *big.Int, name string) error {
	if val.Sign() < 0 {
		log.Warn("ArbOS storage big uint underflowed", "name", name, "value", val)
		val = common.Big0
	} else if val.BitLen() > 256 {
		log.Warn("ArbOS storage big uint overflowed", "name", name, "value", val)
		val = twoToThe256MinusOne
	}
	return sbbu.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

type StorageBackedBigInt struct {
	StorageSlot
}

func (store *Storage) OpenStorageBackedBigInt(offset uint64) StorageBackedBigInt {
	return StorageBackedBigInt{store.NewSlot(offset)}
}

func (sbbi *StorageBackedBigInt) Get() (*big.Int, error) {
	asHash, err := sbbi.StorageSlot.Get()
	if err != nil {
		return nil, err
	}
	asBig := new(big.Int).SetBytes(asHash[:])
	if asBig.Bit(255) != 0 {
		asBig = new(big.Int).Sub(asBig, twoToThe256)
	}
	return asBig, err
}

// Warning: this will panic if it underflows or overflows with a system burner
// SetSaturatingWithWarning is likely better
func (sbbi *StorageBackedBigInt) SetChecked(val *big.Int) error {
	if val.Sign() < 0 {
		val = new(big.Int).Add(val, twoToThe256)
		if val.BitLen() < 256 || val.Sign() <= 0 { // require that it's positive and the top bit is set
			return sbbi.burner.HandleError(fmt.Errorf("underflow in StorageBackedBigInt.Set setting value %v", val))
		}
	} else if val.BitLen() >= 256 {
		return sbbi.burner.HandleError(fmt.Errorf("overflow in StorageBackedBigInt.Set setting value %v", val))
	}
	return sbbi.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

func (sbbi *StorageBackedBigInt) SetSaturatingWithWarning(val *big.Int, name string) error {
	if val.Sign() < 0 {
		origVal := val
		val = new(big.Int).Add(val, twoToThe256)
		if val.BitLen() < 256 || val.Sign() <= 0 { // require that it's positive and the top bit is set
			log.Warn("ArbOS storage big uint underflowed", "name", name, "value", origVal)
			val.Set(twoToThe255)
		}
	} else if val.BitLen() >= 256 {
		log.Warn("ArbOS storage big uint overflowed", "name", name, "value", val)
		val = twoToThe255MinusOne
	}
	return sbbi.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

func (sbbi *StorageBackedBigInt) Set_preVersion7(val *big.Int) error {
	return sbbi.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

func (sbbi *StorageBackedBigInt) SetByUint(val uint64) error {
	return sbbi.StorageSlot.Set(util.UintToHash(val))
}

type StorageBackedAddress struct {
	StorageSlot
}

func (store *Storage) OpenStorageBackedAddress(offset uint64) StorageBackedAddress {
	return StorageBackedAddress{store.NewSlot(offset)}
}

func (sba *StorageBackedAddress) Get() (common.Address, error) {
	value, err := sba.StorageSlot.Get()
	return common.BytesToAddress(value.Bytes()), err
}

func (sba *StorageBackedAddress) Set(val common.Address) error {
	return sba.StorageSlot.Set(util.AddressToHash(val))
}

type StorageBackedAddressOrNil struct {
	StorageSlot
}

var NilAddressRepresentation common.Hash

func init() {
	NilAddressRepresentation = common.BigToHash(new(big.Int).Lsh(big.NewInt(1), 255))
}

func (store *Storage) OpenStorageBackedAddressOrNil(offset uint64) StorageBackedAddressOrNil {
	return StorageBackedAddressOrNil{store.NewSlot(offset)}
}

func (sba *StorageBackedAddressOrNil) Get() (*common.Address, error) {
	asHash, err := sba.StorageSlot.Get()
	if asHash == NilAddressRepresentation || err != nil {
		return nil, err
	} else {
		ret := common.BytesToAddress(asHash.Bytes())
		return &ret, nil
	}
}

func (sba *StorageBackedAddressOrNil) Set(val *common.Address) error {
	if val == nil {
		return sba.StorageSlot.Set(NilAddressRepresentation)
	} else {
		return sba.StorageSlot.Set(common.BytesToHash(val.Bytes()))
	}
}

type StorageBackedBytes struct {
	Storage
}

func (store *Storage) OpenStorageBackedBytes(id []byte) StorageBackedBytes {
	return StorageBackedBytes{
		*store.OpenSubStorage(id),
	}
}

func (sbb *StorageBackedBytes) Get() ([]byte, error) {
	return sbb.Storage.GetBytes()
}

func (sbb *StorageBackedBytes) Set(val []byte) error {
	return sbb.Storage.SetBytes(val)
}

func (sbb *StorageBackedBytes) Clear() error {
	return sbb.Storage.ClearBytes()
}

func (sbb *StorageBackedBytes) Size() (uint64, error) {
	return sbb.Storage.GetBytesSize()
}
