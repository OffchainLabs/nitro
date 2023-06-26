// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"fmt"
	"math/big"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/signature"
)

func OpenWallet(description string, walletConfig *genericconf.WalletConfig, chainId *big.Int) (*bind.TransactOpts, signature.DataSignerFunc, error) {
	if walletConfig.PrivateKey != "" {
		privateKey, err := crypto.HexToECDSA(walletConfig.PrivateKey)
		if err != nil {
			return nil, nil, err
		}
		var txOpts *bind.TransactOpts
		if chainId != nil {
			txOpts, err = bind.NewKeyedTransactorWithChainID(privateKey, chainId)
			if err != nil {
				return nil, nil, err
			}
		}
		signer := func(data []byte) ([]byte, error) {
			return crypto.Sign(data, privateKey)
		}

		return txOpts, signer, nil
	}

	ks := keystore.NewKeyStore(
		walletConfig.Pathname,
		keystore.StandardScryptN,
		keystore.StandardScryptP,
	)

	account, err := openKeystore(ks, description, walletConfig, readPass)
	if err != nil {
		return nil, nil, fmt.Errorf("opening key store: %v", err)
	}
	if walletConfig.OnlyCreateKey {
		log.Info(fmt.Sprintf("Wallet key created with address %s, backup wallet (%s) and remove --%s.wallet.only-create-key to run normally", account.Address.Hex(), walletConfig.Pathname, description))
		return nil, nil, nil
	}

	var txOpts *bind.TransactOpts
	if chainId != nil {
		txOpts, err = bind.NewKeyStoreTransactorWithChainID(ks, *account, chainId)
		if err != nil {
			return nil, nil, err
		}
	}
	signer := func(data []byte) ([]byte, error) {
		return ks.SignHash(*account, data)
	}

	return txOpts, signer, nil
}

func openKeystore(ks *keystore.KeyStore, description string, walletConfig *genericconf.WalletConfig, getPassword func() (string, error)) (*accounts.Account, error) {
	creatingNew := len(ks.Accounts()) == 0
	if creatingNew && !walletConfig.OnlyCreateKey {
		return nil, fmt.Errorf("no wallet exists, re-run with --%s.wallet.only-create-key to create a wallet", description)
	}
	if !creatingNew && walletConfig.OnlyCreateKey {
		return nil, fmt.Errorf("wallet key already created, backup key (%s) and remove --%s.wallet.only-create-key to run normally", walletConfig.Pathname, description)
	}
	passOpt := walletConfig.Password()
	var password string
	if passOpt != nil {
		password = *passOpt
	} else {
		if creatingNew {
			fmt.Print("Enter new account password: ")
		} else {
			fmt.Print("Enter account password: ")
		}
		var err error
		password, err = getPassword()
		if err != nil {
			return nil, err
		}
	}

	if creatingNew {
		a, err := ks.NewAccount(password)
		return &a, err
	}

	var account accounts.Account
	if walletConfig.Account == "" {
		if len(ks.Accounts()) > 1 {
			names := make([]string, 0, len(ks.Accounts()))
			for _, acct := range ks.Accounts() {
				names = append(names, acct.Address.Hex())
			}
			return nil, fmt.Errorf("too many existing accounts, choose one: %s", strings.Join(names, ","))
		}
		account = ks.Accounts()[0]
	} else {
		address := common.HexToAddress(walletConfig.Account)
		var emptyAddress common.Address
		if address == emptyAddress {
			return nil, fmt.Errorf("supplied address is invalid: %s", walletConfig.Account)
		}
		var err error
		account, err = ks.Find(accounts.Account{Address: address})
		if err != nil {
			return nil, err
		}
	}

	if err := ks.Unlock(account, password); err != nil {
		return nil, fmt.Errorf("unlocking the account: %v", err)
	}

	return &account, nil
}

func readPass() (string, error) {
	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	passphrase := string(bytePassword)
	passphrase = strings.TrimSpace(passphrase)
	return passphrase, nil
}
