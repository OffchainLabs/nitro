// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

func GetTransactOptsFromWallet(walletConfig *genericconf.WalletConfig, chainId *big.Int) (*bind.TransactOpts, error) {
	if walletConfig.PrivateKey != "" {
		privateKey, err := crypto.HexToECDSA(walletConfig.PrivateKey)
		if err != nil {
			return nil, err
		}
		return bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	}

	if walletConfig.Pathname == "" {
		return nil, errors.New("keystore path empty")
	}
	l1keystore := keystore.NewKeyStore(walletConfig.Pathname, keystore.StandardScryptN, keystore.StandardScryptP)
	var l1Account accounts.Account
	if walletConfig.Account == "" {
		if len(l1keystore.Accounts()) == 0 {
			return nil, errors.New("keystore empty")
		}
		l1Account = l1keystore.Accounts()[0]
	} else {
		address := common.HexToAddress(walletConfig.Account)
		var err error
		l1Account, err = l1keystore.Find(accounts.Account{Address: address})
		if err != nil {
			return nil, err
		}
	}
	if walletConfig.Password() == nil {
		panic("l2 password not set")
	}
	err := l1keystore.Unlock(l1Account, *walletConfig.Password())
	if err != nil {
		return nil, err
	}
	return bind.NewKeyStoreTransactorWithChainID(l1keystore, l1Account, chainId)
}
