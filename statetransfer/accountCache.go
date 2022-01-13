//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"encoding/json"
	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"os"
	"path/filepath"
)

func getAccountsAndDiff(
	accountAddresses []common.Address,
	accountHashes map[common.Hash]struct{},
	cachePath *string,
) (accountMap map[common.Address]AccountInitializationInfo, missingAddresses []common.Address, err error) {
	if cachePath == nil {
		return nil, accountAddresses, nil
	}
	fullPath := filepath.Join(*cachePath, "accountData")

	_, err = os.Stat(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, accountAddresses, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, accountAddresses, nil
	}
	defer file.Close()
	f := brotli.NewReader(file)

	contents, err := io.ReadAll(f)
	if err != nil {
		return nil, accountAddresses, err
	}

	var fromCache []AccountInitializationInfo
	if err := json.Unmarshal(contents, &fromCache); err != nil {
		return nil, accountAddresses, err
	}

	accountMap = make(map[common.Address]AccountInitializationInfo)
	for _, acct := range fromCache {
		_, exists := accountHashes[acct.ClassicHash]
		if exists {
			accountMap[acct.Addr] = acct
		}
	}

	for _, addr := range accountAddresses {
		_, exists := accountMap[addr]
		if !exists {
			missingAddresses = append(missingAddresses, addr)
		}
	}

	return
}

func flushAccountDataCache(cachePath *string, accountMap map[common.Address]AccountInitializationInfo) error {
	fullPath := filepath.Join(*cachePath, "accountData")
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	f := brotli.NewWriter(file)
	defer func() {
		f.Close()
		file.Close()
	}()

	marshaled, err := json.Marshal(accountMap)
	if err != nil {
		return err
	}

	if _, err := f.Write(marshaled); err != nil {
		return err
	}

	return nil
}
