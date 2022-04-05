// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

func GetTransactOptsFromKeystore(keystorePath, accountAddress, passphrase string, chainId *big.Int) (*bind.TransactOpts, error) {
	if keystorePath == "" {
		return nil, errors.New("keystore path empty")
	}
	l1keystore := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	var l1Account accounts.Account
	if accountAddress == "" {
		if len(l1keystore.Accounts()) == 0 {
			return nil, errors.New("keystore empty")
		}
		l1Account = l1keystore.Accounts()[0]
	} else {
		address := common.HexToAddress(accountAddress)
		var err error
		l1Account, err = l1keystore.Find(accounts.Account{Address: address})
		if err != nil {
			return nil, err
		}
	}
	err := l1keystore.Unlock(l1Account, passphrase)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyStoreTransactorWithChainID(l1keystore, l1Account, chainId)
}
