package util

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SimulatedBackendWrapper is wrapper around SimulatedBackend that implements util.L1Interface
type SimulatedBackendWrapper struct {
	*backends.SimulatedBackend
}

func (s SimulatedBackendWrapper) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	return common.Address{}, nil
}

func (s SimulatedBackendWrapper) BlockNumber(ctx context.Context) (uint64, error) {
	return 0, nil
}
