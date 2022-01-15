//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package blsTable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
)

type BLSTable struct {
	backingStorage *storage.Storage
	byAddress      *storage.Storage
}

var byAddressKey = []byte{0}

func InitializeBLSTable() {
	// no initialization needed
}

func Open(sto *storage.Storage) *BLSTable {
	return &BLSTable{sto, sto.OpenSubStorage(byAddressKey)}
}

func (tab *BLSTable) GetPublicKey(addr common.Address) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	key := common.BytesToHash(append(addr.Bytes(), byte(0)))

	x0, _ := tab.byAddress.Get(key)
	x1, _ := tab.byAddress.Get(util.HashPlusInt(key, 1))
	y0, _ := tab.byAddress.Get(util.HashPlusInt(key, 2))
	y1, err := tab.byAddress.Get(util.HashPlusInt(key, 3))

	return x0.Big(), x1.Big(), y0.Big(), y1.Big(), err
}

func (tab *BLSTable) Register(addr common.Address, x0, x1, y0, y1 *big.Int) error {
	key := common.BytesToHash(append(addr.Bytes(), byte(0)))

	_ = tab.byAddress.Set(key, common.BigToHash(x0))
	_ = tab.byAddress.Set(util.HashPlusInt(key, 1), common.BigToHash(x1))
	_ = tab.byAddress.Set(util.HashPlusInt(key, 2), common.BigToHash(y0))
	return tab.byAddress.Set(util.HashPlusInt(key, 3), common.BigToHash(y1))
}
