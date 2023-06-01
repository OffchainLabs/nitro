package simulated_backend

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Wrapper is wrapper around SimulatedBackend that implements util.L1Interface
type Wrapper struct {
	*backends.SimulatedBackend
}

func (s Wrapper) TransactionSender(_ context.Context, _ *types.Transaction, _ common.Hash, _ uint) (common.Address, error) {
	return common.Address{}, nil
}

func (s Wrapper) BlockNumber(_ context.Context) (uint64, error) {
	return 0, nil
}
