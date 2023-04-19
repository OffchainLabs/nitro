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

func (s SimulatedBackendWrapper) TransactionSender(_ context.Context, _ *types.Transaction, _ common.Hash, _ uint) (common.Address, error) {
	return common.Address{}, nil
}

func (s SimulatedBackendWrapper) BlockNumber(_ context.Context) (uint64, error) {
	return 0, nil
}
