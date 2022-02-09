package util

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

func GetTransactOptsFromKeystore(keystorePath, accoundAddress, passphrase string, chainId *big.Int) (*bind.TransactOpts, error) {
	if keystorePath == "" {
		return nil, errors.New("keystore path empty")
	}
	l1keystore := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	var l1Account accounts.Account
	if accoundAddress == "" {
		if len(l1keystore.Accounts()) == 0 {
			return nil, errors.New("keystore empty")
		}
		l1Account = l1keystore.Accounts()[0]
	} else {
		address := common.HexToAddress(accoundAddress)
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
