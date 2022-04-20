// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func ReadStateFromClassic(ctxIn context.Context, rpcClient *rpc.Client, blockNumber uint64, prevFile, nextBase string, newAPIs bool, blocksOnly bool) error {

	fmt.Println("Initializing")

	ctx, cancel := signal.NotifyContext(ctxIn, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	callopts := &bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Context:     ctx,
	}

	dataHeader := ArbosInitFileContents{
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
	blocksJsonPath := filepath.Join(nextBase, dataHeader.BlocksPath)
	_, err := os.Stat(blocksJsonPath)
	var blockWriter *JsonListWriter
	var blocksCopied int64
	var blockHash common.Hash
	if errors.Is(err, os.ErrNotExist) {
		blockReader, err := reader.GetStoredBlockReader()
		if err != nil {
			return err
		}
		blockWriter, err = NewJsonListWriter(blocksJsonPath, false)
		if err != nil {
			return err
		}
		blocksCopied, blockHash, err = scanAndCopyBlocks(blockReader, blockWriter)
		if err != nil {
			return err
		}
		if err := blockReader.Close(); err != nil {
			return err
		}
	} else if err == nil {
		listReader, err := NewJsonListReader(blocksJsonPath)
		blocksCopied, blockHash, err = scanAndCopyBlocks(&JsonStoredBlockReader{listReader}, nil)
		if err != nil {
			return err
		}
		if err := listReader.Close(); err != nil {
			return err
		}
		blockWriter, err = NewJsonListWriter(blocksJsonPath, true)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	fmt.Println("Reading Blocks")
	if err := fillBlocks(ctx, rpcClient, uint64(blocksCopied), blockNumber, blockHash, blockWriter); err != nil {
		_ = blockWriter.Close()
		return err
	}
	if err := blockWriter.Close(); err != nil {
		return err
	}

	if !blocksOnly {
		fmt.Println("Copying Address Table")
		addressTableReader, err := reader.GetAddressTableReader()
		if err != nil {
			return err
		}
		addressTableWriter, err := NewJsonListWriter(filepath.Join(nextBase, dataHeader.AddressTableContentsPath), false)
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
	} else {
		dataHeader.AddressTableContentsPath = ""
	}

	if newAPIs && !blocksOnly {
		fmt.Println("Retryables")
		retriableWriter, err := NewJsonListWriter(filepath.Join(nextBase, dataHeader.RetryableDataPath), false)
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
		dataHeader.RetryableDataPath = ""
	}

	if !blocksOnly {
		fmt.Println("Accounts")
		accountWriter, err := NewJsonListWriter(filepath.Join(nextBase, dataHeader.AccountsPath), false)
		if err != nil {
			return err
		}
		if newAPIs {
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
	} else {
		dataHeader.AccountsPath = ""
	}

	headerJson, err := json.Marshal(dataHeader)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(nextBase, "header.json"), headerJson, 0600)
}
