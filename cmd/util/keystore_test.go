// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/offchainlabs/nitro/cmd/genericconf"
)

func openTestKeystore(description string, walletConfig *genericconf.WalletConfig, getPassword func() (string, error)) (*keystore.KeyStore, *accounts.Account, error) {
	ks := keystore.NewKeyStore(
		walletConfig.Pathname,
		keystore.LightScryptN,
		keystore.LightScryptP,
	)
	acc, err := openKeystore(ks, description, walletConfig, getPassword)
	return ks, acc, err
}

func createWallet(t *testing.T, pathname string) {
	t.Helper()
	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = pathname
	walletConf.OnlyCreateKey = true
	walletConf.Password = "foo"

	testPassCalled := false
	testPass := func() (string, error) {
		testPassCalled = true
		return "", nil
	}

	if _, _, err := openTestKeystore("test", &walletConf, testPass); err != nil {
		t.Fatalf("openTestKeystore() unexpected error: %v", err)
	}
	if testPassCalled {
		t.Error("password prompted for when it should not have been")
	}
}

func TestNewKeystoreNoCreate(t *testing.T) {
	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = t.TempDir()
	walletConf.OnlyCreateKey = false

	_, _, err := openTestKeystore("test", &walletConf, readPass)
	if err == nil {
		t.Fatalf("should have failed")
	}
	noWalletError := "no wallet exists"
	if !strings.Contains(err.Error(), noWalletError) {
		t.Fatalf("incorrect failure: %v, should have been %s", err, noWalletError)
	}
}

func TestExistingKeystoreNoCreate(t *testing.T) {
	pathname := t.TempDir()

	// Create dummy wallet
	createWallet(t, pathname)

	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = pathname
	walletConf.OnlyCreateKey = true
	walletConf.Password = "foo"

	testPassCalled := false
	testPass := func() (string, error) {
		testPassCalled = true
		return "", nil
	}

	_, _, err := openTestKeystore("test", &walletConf, testPass)
	if err == nil {
		t.Fatalf("should have failed")
	}
	keyAlreadyCreatedError := "wallet key already created"
	if !strings.Contains(err.Error(), keyAlreadyCreatedError) {
		t.Fatalf("incorrect failure: %v, should have been %s", err, keyAlreadyCreatedError)
	}
	if testPassCalled {
		t.Error("password prompted for when it should not have been")
	}
}

func TestNewKeystoreNewPasswordConfig(t *testing.T) {
	createWallet(t, t.TempDir())
}

func TestNewKeystorePromptPasswordTerminal(t *testing.T) {
	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = t.TempDir()
	walletConf.OnlyCreateKey = true
	password := "foo"

	testPassCalled := false
	getPass := func() (string, error) {
		testPassCalled = true
		return password, nil
	}

	if _, _, err := openTestKeystore("test", &walletConf, getPass); err != nil {
		t.Fatalf("openTestKeystore() unexpected error: %v", err)
	}
	if !testPassCalled {
		t.Error("password not prompted for")
	}

	// Unit test doesn't like unflushed output
	fmt.Printf("\n")
}

func TestExistingKeystorePromptPasswordTerminal(t *testing.T) {
	pathname := t.TempDir()

	// Create dummy wallet
	createWallet(t, pathname)

	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = pathname
	walletConf.OnlyCreateKey = false
	password := "foo"

	testPassCalled := false
	testPass := func() (string, error) {
		testPassCalled = true
		return password, nil
	}

	_, _, err := openTestKeystore("test", &walletConf, testPass)
	if err != nil {
		t.Fatalf("should not have have failed")
	}
	if !testPassCalled {
		t.Error("password not prompted for")
	}

	// Unit test doesn't like unflushed output
	fmt.Printf("\n")
}

func TestExistingKeystoreAccountName(t *testing.T) {
	walletConf := genericconf.WalletConfigDefault
	walletConf.Pathname = t.TempDir()
	walletConf.OnlyCreateKey = true
	password := "foo"

	testPassCalled := false
	testPass := func() (string, error) {
		testPassCalled = true
		return password, nil
	}

	if _, _, err := openTestKeystore("test", &walletConf, testPass); err != nil {
		t.Fatalf("openTestKeystore() unexpected error: %v", err)
	}
	if !testPassCalled {
		t.Error("password not prompted for")
	}

	// Get new account
	walletConf.OnlyCreateKey = false
	_, account, err := openTestKeystore("test", &walletConf, testPass)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if account == nil {
		t.Fatal("account missing")
	}

	// Request account by name
	walletConf.Account = account.Address.Hex()
	_, account2, err := openTestKeystore("test", &walletConf, testPass)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if !strings.EqualFold(account2.Address.Hex(), walletConf.Account) {
		t.Fatalf("requested account %s doesn't match returned account %s", walletConf.Account, account2.Address.Hex())
	}

	// Test getting key with invalid address
	walletConf.Account = "junk"
	_, _, err = openTestKeystore("test", &walletConf, testPass)
	if err == nil {
		t.Fatal("should have failed")
	}
	invalidAddressError := "address is invalid"
	keyCreatedError := "wallet key created"
	if !strings.Contains(err.Error(), invalidAddressError) {
		t.Fatalf("incorrect failure: %v, should have been %s", err, keyCreatedError)
	}

	// Test getting key with incorrect address
	walletConf.Account = "0x85d31caC32F0ECd3a978f31d040528B9A219F1C7"
	_, _, err = openTestKeystore("test", &walletConf, testPass)
	if err == nil {
		t.Fatal("should have failed")
	}
	incorrectAddressError := "no key for given address"
	if !strings.Contains(err.Error(), incorrectAddressError) {
		t.Fatalf("incorrect failure: %v, should have been %s", err, keyCreatedError)
	}

	// Unit test doesn't like unflushed output
	fmt.Printf("\n")
}
