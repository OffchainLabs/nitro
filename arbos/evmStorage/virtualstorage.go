package evmStorage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Virtual struct {
	backingStorage T
	uniqueKey      common.Hash
}

func NewVirtual(backingStorage T, uniqueKey common.Hash) T {
	return &Virtual{backingStorage, uniqueKey}
}

func (vs *Virtual) mapKey(key common.Hash) common.Hash {
	return common.BytesToHash(crypto.Keccak256(vs.uniqueKey.Bytes(), key.Bytes()))
}

func (vs *Virtual) Get(key common.Hash) common.Hash {
	return vs.backingStorage.Get(vs.mapKey(key))
}

func (vs *Virtual) Set(key common.Hash, value common.Hash) {
	vs.backingStorage.Set(vs.mapKey(key), value)
}

func (vs *Virtual) Swap(key common.Hash, value common.Hash) common.Hash {
	return vs.backingStorage.Swap(vs.mapKey(key), value)
}
