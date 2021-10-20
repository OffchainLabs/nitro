package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type virtual struct {
	backingStorage Storage
	uniqueKey      common.Hash
}

func NewVirtual(backingStorage Storage, uniqueKey common.Hash) Storage {
	return &virtual{backingStorage, uniqueKey}
}

func (vs *virtual) mapKey(key common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(vs.uniqueKey.Bytes(), key.Bytes()))
}

func (vs *virtual) Get(key common.Hash) common.Hash {
	return vs.backingStorage.Get(vs.mapKey(key))
}

func (vs *virtual) Set(key common.Hash, value common.Hash) {
	vs.backingStorage.Set(vs.mapKey(key), value)
}

func (vs *virtual) Swap(key common.Hash, value common.Hash) common.Hash {
	return vs.backingStorage.Swap(vs.mapKey(key), value)
}
