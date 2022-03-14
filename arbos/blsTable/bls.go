//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package blsTable

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/blsSignatures"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
)

type BLSTable struct {
	backingStorage         *storage.Storage
	legacyAddressSet       *addressSet.AddressSet
	legacyTableByAddress   *storage.Storage
	bls12381AddressSet     *addressSet.AddressSet
	bls12381TableByAddress *storage.Storage
}

var (
	legacyAddressSetKey       = []byte{0}
	legacyTableByAddressKey   = []byte{1}
	bls12381AddressSetKey     = []byte{2}
	bls12381TableByAddressKey = []byte{3}
)

func InitializeBLSTable(sto *storage.Storage) error {
	err := addressSet.Initialize(sto.OpenSubStorage(legacyAddressSetKey))
	if err != nil {
		return err
	}
	return addressSet.Initialize(sto.OpenSubStorage(bls12381AddressSetKey))
}

func Open(sto *storage.Storage) *BLSTable {
	return &BLSTable{
		sto,
		addressSet.OpenAddressSet(sto.OpenSubStorage(legacyAddressSetKey)),
		sto.OpenSubStorage(legacyTableByAddressKey),
		addressSet.OpenAddressSet(sto.OpenSubStorage(bls12381AddressSetKey)),
		sto.OpenSubStorage(bls12381TableByAddressKey),
	}
}

func (tab *BLSTable) GetLegacyPublicKey(addr common.Address) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	isMember, err := tab.legacyAddressSet.IsMember(addr)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if !isMember {
		return nil, nil, nil, nil, ethereum.NotFound
	}

	key := common.BytesToHash(append(addr.Bytes(), byte(0)))

	x0, _ := tab.legacyTableByAddress.Get(key)
	x1, _ := tab.legacyTableByAddress.Get(util.HashPlusInt(key, 1))
	y0, _ := tab.legacyTableByAddress.Get(util.HashPlusInt(key, 2))
	y1, err := tab.legacyTableByAddress.Get(util.HashPlusInt(key, 3))

	return x0.Big(), x1.Big(), y0.Big(), y1.Big(), err
}

func (tab *BLSTable) RegisterLegacyPublicKey(addr common.Address, x0, x1, y0, y1 *big.Int) error {
	key := common.BytesToHash(append(addr.Bytes(), byte(0)))

	_ = tab.legacyTableByAddress.Set(key, common.BigToHash(x0))
	_ = tab.legacyTableByAddress.Set(util.HashPlusInt(key, 1), common.BigToHash(x1))
	_ = tab.legacyTableByAddress.Set(util.HashPlusInt(key, 2), common.BigToHash(y0))
	_ = tab.legacyTableByAddress.Set(util.HashPlusInt(key, 3), common.BigToHash(y1))
	return tab.legacyAddressSet.Add(addr)
}

func (tab *BLSTable) RegisterBLS12381PublicKey(addr common.Address, key blsSignatures.PublicKey) error {
	if err := tab.bls12381AddressSet.Add(addr); err != nil {
		return err
	}

	sbBytes := tab.bls12381TableByAddress.OpenStorageBackedBytes(addr.Bytes())
	return sbBytes.SetBytes(blsSignatures.PublicKeyToBytes(key))
}

func (tab *BLSTable) GetBLS12381PublicKey(addr common.Address) (blsSignatures.PublicKey, error) {
	isMember, err := tab.bls12381AddressSet.IsMember(addr)
	if err != nil {
		return blsSignatures.PublicKey{}, err
	}
	if !isMember {
		return blsSignatures.PublicKey{}, ethereum.NotFound
	}

	sbBytes := tab.bls12381TableByAddress.OpenStorageBackedBytes(addr.Bytes())
	buf, err := sbBytes.GetBytes()
	if err != nil {
		return blsSignatures.PublicKey{}, err
	}
	return blsSignatures.PublicKeyFromBytes(buf, true)
}
