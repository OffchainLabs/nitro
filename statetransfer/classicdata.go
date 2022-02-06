//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"context"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func ReadStateFromClassic(ctx context.Context, rpcClient *rpc.Client, blockNumber uint64, prevData io.Reader, curData io.Writer, oldAPIs bool) error {

	fmt.Println("Initializing")

	callopts := &bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Context:     ctx,
	}

	ethClient := ethclient.NewClient(rpcClient)

	var reader *IterativeJsonReader
	if prevData != nil {
		reader = NewIterativeJsonReader(prevData)
	}
	writer := NewIterativeJsonWriter(curData)
	updater := &IterativeJsonUpdater{reader, writer}

	err := updater.OpenTopLevel()
	if err != nil {
		return err
	}

	err = updater.StartSubList("Blocks")
	if err != nil {
		return err
	}

	fmt.Println("Copying Blocks")

	blocksCopied, blockHash, err := scanAndCopyBlocks(reader, writer)
	if err != nil {
		return err
	}

	fmt.Println("Reading Blocks")

	err = fillBlocks(ctx, rpcClient, uint64(blocksCopied), blockNumber, blockHash, writer)
	if err != nil {
		return err
	}

	err = updater.CloseList()
	if err != nil {
		return err
	}

	fmt.Println("Address Table")

	err = updater.StartSubList("AddressTableContents")
	if err != nil {
		return err
	}

	prevLength, lastAddress, err := scanAndCopyAddressTable(reader, writer)
	if err != nil {
		return err
	}

	err = verifyAndFillAddressTable(ethClient, callopts, prevLength, lastAddress, writer)
	if err != nil {
		return err
	}

	err = updater.CloseList()
	if err != nil {
		return err
	}

	fmt.Println("Retriables")

	err = updater.StartSubList("RetryableData")
	if err != nil {
		return err
	}

	err = skipRetriables(reader)
	if err != nil {
		return err
	}

	classicArbRetryableTx, err := openClassicArbRetryableTx(ethClient)
	if err != nil {
		return err
	}
	if !oldAPIs {
		retryables, err := getRetryables(classicArbRetryableTx, callopts)
		if err != nil {
			return err
		}
		for _, retriable := range retryables {
			err := writer.AddElement(retriable)
			if err != nil {
				return err
			}
		}
	}
	err = updater.CloseList()
	if err != nil {
		return err
	}

	fmt.Println("Accounts")

	err = updater.StartSubList("Accounts")
	if err != nil {
		return err
	}

	classicArbosTest, err := openClassicArbosTest(ethClient)
	if err != nil {
		return err
	}

	if !oldAPIs {
		accountHashes, err := getAccountHashesAsMap(classicArbosTest, callopts)
		if err != nil {
			return err
		}
		foundAddresses, err := copyStillValidAccounts(reader, writer, accountHashes)
		if err != nil {
			return err
		}
		accounts, err := getAccountMap(classicArbosTest, callopts)
		if err != nil {
			return err
		}
		err = fillAccounts(writer, classicArbosTest, callopts, accounts, foundAddresses)
		if err != nil {
			return err
		}
	} else {
		err = skipAccounts(reader)
		if err != nil {
			return err
		}
		err = fillAccountsOld(writer, *ethClient, callopts, tempAccountList)
		if err != nil {
			return err
		}
	}

	err = updater.CloseList()
	if err != nil {
		return err
	}

	return updater.CloseTopLevel()
}
