// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func ReadStateFromClassic(ctx context.Context, rpcClient *rpc.Client, blockNumber uint64, prevFile, nextBase string, oldAPIs bool) error {

	fmt.Println("Initializing")

	callopts := &bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Context:     ctx,
	}

	DataHeader := ArbosInitFileContents{
		BlocksPath:               "blocks.json",
		AddressTableContentsPath: "addresstable.json",
		RetryableDataPath:        "retriables.json",
		AccountsPath:             "accounts.json",
	}

	ethClient := ethclient.NewClient(rpcClient)

	var reader InitDataReader
	if prevFile == "" {
		reader = NewMemoryInitDataReader(&ArbosInitializationInfo{})
	} else {
		var err error
		reader, err = NewJsonInitDataReader(prevFile)
		if err != nil {
			return err
		}
	}

	fmt.Println("Copying Blocks")
	blockReader, err := reader.GetStoredBlockReader()
	if err != nil {
		return err
	}
	blockWriter, err := NewJsonListWriter(path.Join(nextBase, DataHeader.BlocksPath))
	if err != nil {
		return err
	}
	blocksCopied, blockHash, err := scanAndCopyBlocks(blockReader, blockWriter)
	if err != nil {
		return err
	}
	fmt.Println("Reading Blocks")
	if err := fillBlocks(ctx, rpcClient, uint64(blocksCopied), blockNumber, blockHash, blockWriter); err != nil {
		return err
	}
	if err := blockWriter.Close(); err != nil {
		return err
	}
	if err := blockReader.Close(); err != nil {
		return err
	}

	fmt.Println("Copying Address Table")
	addressTableReader, err := reader.GetAddressTableReader()
	if err != nil {
		return err
	}
	addressTableWriter, err := NewJsonListWriter(path.Join(nextBase, DataHeader.AddressTableContentsPath))
	if err != nil {
		return err
	}

	prevLength, lastAddress, err := scanAndCopyAddressTable(addressTableReader, addressTableWriter)
	if err != nil {
		return err
	}
	fmt.Println("Reading Address Table")
	if err := verifyAndFillAddressTable(ethClient, callopts, prevLength, lastAddress, addressTableWriter); err != nil {
		return err
	}
	if err := addressTableWriter.Close(); err != nil {
		return err
	}
	if err := addressTableReader.Close(); err != nil {
		return err
	}

	fmt.Println("Retriables")
	if !oldAPIs {
		retriableWriter, err := NewJsonListWriter(path.Join(nextBase, DataHeader.RetryableDataPath))
		if err != nil {
			return err
		}
		classicArbRetryableTx, err := openClassicArbRetryableTx(ethClient)
		if err != nil {
			return err
		}

		retryables, err := getRetryables(classicArbRetryableTx, callopts)
		if err != nil {
			return err
		}
		for _, retriable := range retryables {
			err := retriableWriter.Write(retriable)
			if err != nil {
				return err
			}
		}
		if err := retriableWriter.Close(); err != nil {
			return err
		}
	} else {
		DataHeader.RetryableDataPath = ""
	}

	fmt.Println("Accounts")
	accountWriter, err := NewJsonListWriter(path.Join(nextBase, DataHeader.AccountsPath))
	if err != nil {
		return err
	}
	if !oldAPIs {
		classicArbosTest, err := openClassicArbosTest(ethClient)
		if err != nil {
			return err
		}
		accountReader, err := reader.GetAccountDataReader()
		if err != nil {
			return err
		}
		accountHashes, err := getAccountHashesAsMap(classicArbosTest, callopts)
		if err != nil {
			return err
		}
		foundAddresses, err := copyStillValidAccounts(accountReader, accountWriter, accountHashes)
		if err != nil {
			return err
		}
		accounts, err := getAccountMap(classicArbosTest, callopts)
		if err != nil {
			return err
		}
		if err := fillAccounts(accountWriter, classicArbosTest, callopts, accounts, foundAddresses); err != nil {
			return err
		}
		if err := accountReader.Close(); err != nil {
			return err
		}
	} else {
		err := fillAccountsOld(accountWriter, *ethClient, callopts, tempAccountList)
		if err != nil {
			return err
		}
	}
	if err := accountWriter.Close(); err != nil {
		return err
	}

	headerJson, err := json.Marshal(DataHeader)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(nextBase, "header.json"), headerJson, 0600)
}
