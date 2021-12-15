//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/solgen/go/classicgen"
	"math/big"
)

func openClassicArbAddressTable(client *ethclient.Client) (*classicgen.ArbAddressTableCaller, error) {
	return classicgen.NewArbAddressTableCaller(common.BigToAddress(big.NewInt(ArbAddressTableAsInt)), client)
}

func getAddressTableContents(caller *classicgen.ArbAddressTableCaller, callopts *bind.CallOpts) ([]common.Address, error) {
	ret := []common.Address{}
	size, err := caller.Size(callopts)
	if err != nil {
		return nil, err
	}
	for i := int64(0); i < size.Int64(); i++ {
		addr, err := caller.LookupIndex(callopts, big.NewInt(i))
		if err != nil {
			return nil, err
		}
		ret = append(ret, addr)
	}
	return ret, nil
}
