// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/solgen/go/classicgen"
	concurrently "github.com/tejzpr/ordered-concurrently/v3"
)

var ArbosAddressTable = common.HexToAddress("0x0000000000000000000000000000000000000066")

func openClassicArbAddressTable(client *ethclient.Client) (*classicgen.ArbAddressTableCaller, error) {
	return classicgen.NewArbAddressTableCaller(ArbosAddressTable, client)
}

func scanAndCopyAddressTable(reader AddressReader, writer *JsonListWriter) (uint64, common.Address, error) {
	length := uint64(0)
	address := &common.Address{}
	for reader.More() {
		var err error
		address, err = reader.GetNext()
		if err != nil {
			return length, common.Address{}, err
		}
		err = writer.Write(address)
		if err != nil {
			return length, common.Address{}, err
		}
		AddressSeen(*address)
		length += 1
	}
	return length, *address, nil
}

type addressQuery struct {
	classicArbAddressTable *classicgen.ArbAddressTableCaller
	callopts               *bind.CallOpts
	cIndex                 int64
}

type addressQueryResult struct {
	account common.Address
	cIndex  int64
	err     error
}

func (q addressQuery) Run(ctx context.Context) interface{} {
	addr, err := q.classicArbAddressTable.LookupIndex(q.callopts, big.NewInt(q.cIndex))
	return addressQueryResult{addr, q.cIndex, err}
}

func verifyAndFillAddressTable(ethClient *ethclient.Client, callopts *bind.CallOpts, prevLength uint64, lastAddress common.Address, writer *JsonListWriter) error {
	classicArbAddressTable, err := openClassicArbAddressTable(ethClient)
	if err != nil {
		return err
	}
	if prevLength > 0 {
		// sanity test for reorgs, etc.. assume all is o.k. if last is o.k.
		lastIndex := big.NewInt(int64(prevLength) - 1)
		foundAddress, err := classicArbAddressTable.LookupIndex(callopts, lastIndex)
		if err != nil {
			return err
		}
		if foundAddress != lastAddress {
			return fmt.Errorf("addresstable index %v expected %s found %s", lastIndex, lastAddress, foundAddress)
		}
	}

	numAddresses, err := classicArbAddressTable.Size(callopts)
	if err != nil {
		return fmt.Errorf("addresstable.Size error: %w", err)
	}
	numAddressesInt := numAddresses.Int64()
	if (!numAddresses.IsInt64()) || numAddressesInt < int64(prevLength) {
		return fmt.Errorf("addresstable size %v expected at least %v", numAddresses, prevLength)
	}
	fmt.Println("current Num of addresses ", numAddresses)

	inputChan := make(chan concurrently.WorkFunction)
	output := concurrently.Process(callopts.Context, inputChan, &concurrently.Options{PoolSize: parallelQueries, OutChannelBuffer: parallelQueries})
	go func() {
		for cIndex := int64(prevLength); cIndex < numAddressesInt; cIndex++ {
			if cIndex%(numAddressesInt/10) == 0 {
				// give the node a bit of time to recover
				time.Sleep(time.Second)
			}
			inputChan <- addressQuery{classicArbAddressTable, callopts, cIndex}
		}
		close(inputChan)
	}()
	for out := range output {
		res, ok := out.Value.(addressQueryResult)
		if !ok {
			return errors.New("unexpected result type from address query")
		}
		if res.err != nil {
			return res.err
		}
		completed := res.cIndex + 1
		totalAddresses := numAddressesInt
		if completed%10 == 0 {
			fmt.Printf("\rRead address %v/%v (%.2f%%)", completed, totalAddresses, 100*float64(completed)/float64(totalAddresses))
		}
		err = writer.Write(res.account)
		if err != nil {
			return err
		}
		AddressSeen(res.account)
	}
	fmt.Printf("\rDone reading addresses!                    \n")
	return nil
}
