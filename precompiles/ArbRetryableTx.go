//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ArbRetryableTx struct{}

func (con ArbRetryableTx) Cancel(caller common.Address, st *state.StateDB, ticketId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) CancelGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetBeneficiary(
	caller common.Address,
	st *state.StateDB,
	ticketId [32]byte,
) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetBeneficiaryGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetKeepalivePrice(
	caller common.Address,
	st *state.StateDB,
	ticketId [32]byte,
) (*big.Int, *big.Int, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetKeepalivePriceGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetLifetime(caller common.Address, st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetLifetimeGasCost() uint64 {
	return 0
}

func (con ArbRetryableTx) GetSubmissionPrice(
	caller common.Address,
	st *state.StateDB,
	calldataSize *big.Int,
) (*big.Int, *big.Int, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetSubmissionPriceGasCost(calldataSize *big.Int) uint64 {
	return 0
}

func (con ArbRetryableTx) GetTimeout(caller common.Address, st *state.StateDB, ticketId [32]byte) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetTimeoutGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) Keepalive(
	caller common.Address,
	st *state.StateDB,
	value *big.Int,
	ticketId [32]byte,
) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) KeepaliveGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) Redeem(caller common.Address, st *state.StateDB, txId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) RedeemGasCost(txId [32]byte) uint64 {
	return 0
}
