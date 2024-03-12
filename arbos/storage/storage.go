// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package storage

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
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
	hashCache  *lru.Cache[string, []byte]
}

const StorageReadCost = params.SloadGasEIP2200
const StorageWriteCost = params.SstoreSetGasEIP2200
const StorageWriteZeroCost = params.SstoreResetGasEIP2200
const StorageCodeHashCost = params.ColdAccountAccessCostEIP2929

const storageKeyCacheSize = 1024

var storageHashCache = lru.NewCache[string, []byte](storageKeyCacheSize)
var cacheFullLogged atomic.Bool

// NewGeth uses a Geth database to create an evm key-value store
func NewGeth(statedb vm.StateDB, burner burn.Burner) *Storage {
	account := common.HexToAddress("0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	statedb.SetNonce(account, 1) // setting the nonce ensures Geth won't treat ArbOS as empty
	return &Storage{
		account:    account,
		db:         statedb,
		storageKey: []byte{},
		burner:     burner,
		hashCache:  storageHashCache,
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
func (s *Storage) mapAddress(key common.Hash) common.Hash {
	keyBytes := key.Bytes()
	boundary := common.HashLength - 1
	mapped := make([]byte, 0, common.HashLength)
	mapped = append(mapped, s.cachedKeccak(s.storageKey, keyBytes[:boundary])[:boundary]...)
	mapped = append(mapped, keyBytes[boundary])
	return common.BytesToHash(mapped)
}

func writeCost(value common.Hash) uint64 {
	if value == (common.Hash{}) {
		return StorageWriteZeroCost
	}
	return StorageWriteCost
}

func (s *Storage) Account() common.Address {
	return s.account
}

func (s *Storage) Get(key common.Hash) (common.Hash, error) {
	err := s.burner.Burn(StorageReadCost)
	if err != nil {
		return common.Hash{}, err
	}
	if info := s.burner.TracingInfo(); info != nil {
		info.RecordStorageGet(key)
	}
	return s.db.GetState(s.account, s.mapAddress(key)), nil
}

func (s *Storage) GetStorageSlot(key common.Hash) common.Hash {
	return s.mapAddress(key)
}

func (s *Storage) GetUint64(key common.Hash) (uint64, error) {
	value, err := s.Get(key)
	return value.Big().Uint64(), err
}

func (s *Storage) GetByUint64(key uint64) (common.Hash, error) {
	return s.Get(util.UintToHash(key))
}

func (s *Storage) GetUint64ByUint64(key uint64) (uint64, error) {
	return s.GetUint64(util.UintToHash(key))
}

func (s *Storage) GetUint32(key common.Hash) (uint32, error) {
	value, err := s.Get(key)
	return uint32(value.Big().Uint64()), err
}

func (s *Storage) Set(key common.Hash, value common.Hash) error {
	if s.burner.ReadOnly() {
		log.Error("Read-only burner attempted to mutate state", "key", key, "value", value)
		return vm.ErrWriteProtection
	}
	err := s.burner.Burn(writeCost(value))
	if err != nil {
		return err
	}
	if info := s.burner.TracingInfo(); info != nil {
		info.RecordStorageSet(key, value)
	}
	s.db.SetState(s.account, s.mapAddress(key), value)
	return nil
}

func (s *Storage) SetUint64(key common.Hash, value uint64) error {
	return s.Set(key, util.UintToHash(value))
}

func (s *Storage) SetByUint64(key uint64, value common.Hash) error {
	return s.Set(util.UintToHash(key), value)
}

func (s *Storage) SetUint64ByUint64(key uint64, value uint64) error {
	return s.Set(util.UintToHash(key), util.UintToHash(value))
}

func (s *Storage) SetUint32(key common.Hash, value uint32) error {
	return s.Set(key, util.UintToHash(uint64(value)))
}

func (s *Storage) SetByUint32(key uint32, value common.Hash) error {
	return s.Set(util.UintToHash(uint64(key)), value)
}

func (s *Storage) Clear(key common.Hash) error {
	return s.Set(key, common.Hash{})
}

func (s *Storage) ClearByUint64(key uint64) error {
	return s.Set(util.UintToHash(key), common.Hash{})
}

func (s *Storage) Swap(key common.Hash, newValue common.Hash) (common.Hash, error) {
	oldValue, err := s.Get(key)
	if err != nil {
		return common.Hash{}, err
	}
	return oldValue, s.Set(key, newValue)
}

func (s *Storage) OpenCachedSubStorage(id []byte) *Storage {
	return &Storage{
		account:    s.account,
		db:         s.db,
		storageKey: s.cachedKeccak(s.storageKey, id),
		burner:     s.burner,
		hashCache:  storageHashCache,
	}
}
func (s *Storage) OpenSubStorage(id []byte) *Storage {
	return &Storage{
		account:    s.account,
		db:         s.db,
		storageKey: s.cachedKeccak(s.storageKey, id),
		burner:     s.burner,
		hashCache:  nil,
	}
}

// Returns shallow copy of Storage that won't use storage key hash cache.
// The storage space represented by the returned Storage is kept the same.
func (s *Storage) WithoutCache() *Storage {
	return &Storage{
		account:    s.account,
		db:         s.db,
		storageKey: s.storageKey,
		burner:     s.burner,
		hashCache:  nil,
	}
}

func (s *Storage) SetBytes(b []byte) error {
	err := s.ClearBytes()
	if err != nil {
		return err
	}
	err = s.SetUint64ByUint64(0, uint64(len(b)))
	if err != nil {
		return err
	}
	offset := uint64(1)
	for len(b) >= 32 {
		err = s.SetByUint64(offset, common.BytesToHash(b[:32]))
		if err != nil {
			return err
		}
		b = b[32:]
		offset++
	}
	return s.SetByUint64(offset, common.BytesToHash(b))
}

func (s *Storage) GetBytes() ([]byte, error) {
	bytesLeft, err := s.GetUint64ByUint64(0)
	if err != nil {
		return nil, err
	}
	ret := []byte{}
	offset := uint64(1)
	for bytesLeft >= 32 {
		next, err := s.GetByUint64(offset)
		if err != nil {
			return nil, err
		}
		ret = append(ret, next.Bytes()...)
		bytesLeft -= 32
		offset++
	}
	next, err := s.GetByUint64(offset)
	if err != nil {
		return nil, err
	}
	ret = append(ret, next.Bytes()[32-bytesLeft:]...)
	return ret, nil
}

func (s *Storage) GetBytesSize() (uint64, error) {
	return s.GetUint64ByUint64(0)
}

func (s *Storage) ClearBytes() error {
	bytesLeft, err := s.GetUint64ByUint64(0)
	if err != nil {
		return err
	}
	offset := uint64(1)
	for bytesLeft > 0 {
		err := s.ClearByUint64(offset)
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
	return s.ClearByUint64(0)
}

func (s *Storage) GetCodeHash(address common.Address) (common.Hash, error) {
	err := s.burner.Burn(StorageCodeHashCost)
	if err != nil {
		return common.Hash{}, err
	}
	return s.db.GetCodeHash(address), nil
}

func (s *Storage) Burner() burn.Burner {
	return s.burner // not public because these should never be changed once set
}

func (s *Storage) Keccak(data ...[]byte) ([]byte, error) {
	byteCount := 0
	for _, part := range data {
		byteCount += len(part)
	}
	cost := 30 + 6*arbmath.WordsForBytes(uint64(byteCount))
	if err := s.burner.Burn(cost); err != nil {
		return nil, err
	}
	return crypto.Keccak256(data...), nil
}

func (s *Storage) KeccakHash(data ...[]byte) (common.Hash, error) {
	bytes, err := s.Keccak(data...)
	return common.BytesToHash(bytes), err
}

// Returns crypto.Keccak256 result for the given data
// If available the result is taken from hash cache
// otherwise crypto.Keccak256 is executed and its result is added to the cache and returned
// note: the method doesn't burn gas, as it's only intended for generating storage subspace keys and mapping slot addresses
// note: returned slice is not thread-safe
func (s *Storage) cachedKeccak(data ...[]byte) []byte {
	if s.hashCache == nil {
		return crypto.Keccak256(data...)
	}
	keyString := string(bytes.Join(data, []byte{}))
	if hash, wasCached := s.hashCache.Get(keyString); wasCached {
		return hash
	}
	hash := crypto.Keccak256(data...)
	evicted := s.hashCache.Add(keyString, hash)
	if evicted && cacheFullLogged.CompareAndSwap(false, true) {
		log.Warn("Hash cache full, we didn't expect that. Some non-static storage keys may fill up the cache.")
	}
	return hash
}

type StorageSlot struct {
	account common.Address
	db      vm.StateDB
	slot    common.Hash
	burner  burn.Burner
}

func (s *Storage) NewSlot(offset uint64) StorageSlot {
	return StorageSlot{s.account, s.db, s.mapAddress(util.UintToHash(offset)), s.burner}
}

func (ss *StorageSlot) Get() (common.Hash, error) {
	err := ss.burner.Burn(StorageReadCost)
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
	err := ss.burner.Burn(writeCost(value))
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

func (s *Storage) OpenStorageBackedInt64(offset uint64) StorageBackedInt64 {
	return StorageBackedInt64{s.NewSlot(offset)}
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

func (s *Storage) OpenStorageBackedBips(offset uint64) StorageBackedBips {
	return StorageBackedBips{StorageBackedInt64{s.NewSlot(offset)}}
}

func (sbu *StorageBackedBips) Get() (arbmath.Bips, error) {
	value, err := sbu.backing.Get()
	return arbmath.Bips(value), err
}

func (sbu *StorageBackedBips) Set(bips arbmath.Bips) error {
	return sbu.backing.Set(int64(bips))
}

// StorageBackedUBips represents an unsigned number of basis points
type StorageBackedUBips struct {
	backing StorageBackedUint64
}

func (s *Storage) OpenStorageBackedUBips(offset uint64) StorageBackedUBips {
	return StorageBackedUBips{StorageBackedUint64{s.NewSlot(offset)}}
}

func (sbu *StorageBackedUBips) Get() (arbmath.UBips, error) {
	value, err := sbu.backing.Get()
	return arbmath.UBips(value), err
}

func (sbu *StorageBackedUBips) Set(bips arbmath.UBips) error {
	return sbu.backing.Set(bips.Uint64())
}

type StorageBackedUint16 struct {
	StorageSlot
}

func (s *Storage) OpenStorageBackedUint16(offset uint64) StorageBackedUint16 {
	return StorageBackedUint16{s.NewSlot(offset)}
}

func (sbu *StorageBackedUint16) Get() (uint16, error) {
	raw, err := sbu.StorageSlot.Get()
	big := raw.Big()
	if !big.IsUint64() || big.Uint64() > math.MaxUint16 {
		panic("expected uint16 compatible value in storage")
	}
	return uint16(big.Uint64()), err
}

func (sbu *StorageBackedUint16) Set(value uint16) error {
	bigValue := new(big.Int).SetUint64(uint64(value))
	return sbu.StorageSlot.Set(common.BigToHash(bigValue))
}

type StorageBackedUint24 struct {
	StorageSlot
}

func (s *Storage) OpenStorageBackedUint24(offset uint64) StorageBackedUint24 {
	return StorageBackedUint24{s.NewSlot(offset)}
}

func (sbu *StorageBackedUint24) Get() (arbmath.Uint24, error) {
	raw, err := sbu.StorageSlot.Get()
	value := arbmath.BigToUint24OrPanic(raw.Big())
	return value, err
}

func (sbu *StorageBackedUint24) Set(value arbmath.Uint24) error {
	return sbu.StorageSlot.Set(common.BigToHash(value.ToBig()))
}

type StorageBackedUint32 struct {
	StorageSlot
}

func (s *Storage) OpenStorageBackedUint32(offset uint64) StorageBackedUint32 {
	return StorageBackedUint32{s.NewSlot(offset)}
}

func (sbu *StorageBackedUint32) Get() (uint32, error) {
	raw, err := sbu.StorageSlot.Get()
	big := raw.Big()
	if !big.IsUint64() || big.Uint64() > math.MaxUint32 {
		panic("expected uint32 compatible value in storage")
	}
	return uint32(big.Uint64()), err
}

func (sbu *StorageBackedUint32) Set(value uint32) error {
	bigValue := new(big.Int).SetUint64(uint64(value))
	return sbu.StorageSlot.Set(common.BigToHash(bigValue))
}

type StorageBackedUint64 struct {
	StorageSlot
}

func (s *Storage) OpenStorageBackedUint64(offset uint64) StorageBackedUint64 {
	return StorageBackedUint64{s.NewSlot(offset)}
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

func (s *Storage) OpenStorageBackedBigUint(offset uint64) StorageBackedBigUint {
	return StorageBackedBigUint{s.NewSlot(offset)}
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
	}
	if val.BitLen() > 256 {
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

func (s *Storage) OpenStorageBackedBigInt(offset uint64) StorageBackedBigInt {
	return StorageBackedBigInt{s.NewSlot(offset)}
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

func (s *Storage) OpenStorageBackedAddress(offset uint64) StorageBackedAddress {
	return StorageBackedAddress{s.NewSlot(offset)}
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

func (s *Storage) OpenStorageBackedAddressOrNil(offset uint64) StorageBackedAddressOrNil {
	return StorageBackedAddressOrNil{s.NewSlot(offset)}
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
	}
	return sba.StorageSlot.Set(common.BytesToHash(val.Bytes()))
}

type StorageBackedBytes struct {
	Storage
}

func (s *Storage) OpenStorageBackedBytes(id []byte) StorageBackedBytes {
	return StorageBackedBytes{
		*s.OpenSubStorage(id),
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
