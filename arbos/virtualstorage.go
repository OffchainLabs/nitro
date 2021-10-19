package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type virtualStorage struct {
	backingStorage EvmStorage
	uniqueKey      common.Hash
}

func NewVirtualStorage(backingStorage EvmStorage, uniqueKey common.Hash) EvmStorage {
	return &virtualStorage{backingStorage, uniqueKey}
}

func (vs *virtualStorage) mapKey(key common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(vs.uniqueKey.Bytes(), key.Bytes()))
}

func (vs *virtualStorage) Get(key common.Hash) common.Hash {
	return vs.backingStorage.Get(vs.mapKey(key))
}

func (vs *virtualStorage) Set(key common.Hash, value common.Hash) {
	vs.backingStorage.Set(vs.mapKey(key), value)
}

func (vs *virtualStorage) Swap(key common.Hash, value common.Hash) common.Hash {
	return vs.backingStorage.Swap(vs.mapKey(key), value)
}
