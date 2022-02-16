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

var ArbTransferListNames = []string{"Blocks", "AddressTableContents", "RetryableData", "Accounts"}

func ReadStateFromClassic(ctx context.Context, rpcClient *rpc.Client, blockNumber uint64, prevData io.Reader, curData io.Writer, oldAPIs bool) error {

	fmt.Println("Initializing")

	callopts := &bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: new(big.Int).SetUint64(blockNumber),
		Context:     ctx,
	}

	ethClient := ethclient.NewClient(rpcClient)

	var reader *JsonMultiListReader
	if prevData != nil {
		reader = NewJsonMultiListReader(prevData, ArbTransferListNames)
	}
	writer := NewJsonMultiListWriter(curData, ArbTransferListNames)
	updater := &JsonMultiListUpdater{reader, writer}

	if err := updater.OpenTopLevel(); err != nil {
		return err
	}

	fmt.Println("Copying Blocks")
	if listname, err := updater.NextList(); err != nil || listname != "Blocks" {
		return fmt.Errorf("expected Blocks, found: %v, err: %w", listname, err)
	}
	blocksCopied, blockHash, err := scanAndCopyBlocks(reader, writer)
	if err != nil {
		return err
	}
	fmt.Println("Reading Blocks")
	if err := fillBlocks(ctx, rpcClient, uint64(blocksCopied), blockNumber, blockHash, writer); err != nil {
		return err
	}
	if err := updater.CloseList(); err != nil {
		return err
	}

	fmt.Println("Copying Address Table")
	if listname, err := updater.NextList(); err != nil || listname != "AddressTableContents" {
		return fmt.Errorf("expected Blocks, found: %v, err: %w", listname, err)
	}
	prevLength, lastAddress, err := scanAndCopyAddressTable(reader, writer)
	if err != nil {
		return err
	}
	fmt.Println("Reading Address Table")
	if err := verifyAndFillAddressTable(ethClient, callopts, prevLength, lastAddress, writer); err != nil {
		return err
	}
	if err := updater.CloseList(); err != nil {
		return err
	}

	fmt.Println("Retriables")
	if listname, err := updater.NextList(); err != nil || listname != "RetryableData" {
		return fmt.Errorf("expected Blocks, found: %v, err: %w", listname, err)
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
	if err := updater.CloseList(); err != nil {
		return err
	}

	fmt.Println("Accounts")
	if listname, err := updater.NextList(); err != nil || listname != "Accounts" {
		return fmt.Errorf("expected Blocks, found: %v, err: %w", listname, err)
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
		if err := fillAccounts(writer, classicArbosTest, callopts, accounts, foundAddresses); err != nil {
			return err
		}
	} else {
		if err := skipAccounts(reader); err != nil {
			return err
		}
		if err := fillAccountsOld(writer, *ethClient, callopts, tempAccountList); err != nil {
			return err
		}
	}
	if err := updater.CloseList(); err != nil {
		return err
	}

	return updater.CloseTopLevel()
}
