package util

import (
	"flag"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/rpc"
)

func GetFinalizedCallOpts(opts *bind.CallOpts) *bind.CallOpts {
	if opts == nil {
		opts = &bind.CallOpts{}
	}
	// If we are running tests, we want to use the latest block number since
	// simulated backends only support the latest block number.
	if flag.Lookup("test.v") != nil {
		return opts
	}
	opts.BlockNumber = big.NewInt(int64(rpc.FinalizedBlockNumber))
	return opts
}

func GetFinalizedBlockNumber() *big.Int {
	// If we are running tests, we want to use the latest block number since
	// simulated backends only support the latest block number.
	if flag.Lookup("test.v") != nil {
		return nil
	}
	return big.NewInt(int64(rpc.FinalizedBlockNumber))
}
