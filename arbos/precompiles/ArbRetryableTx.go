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

func (con ArbRetryableTx) Cancel(st *state.StateDB, ticketId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) GetBeneficiary(st *state.StateDB, ticketId [32]byte) (common.Address, error) {
	return common.Address{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetKeepalivePrice(st *state.StateDB, ticketId [32]byte) (*big.Int, *big.Int, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetLifetime(st *state.StateDB) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetSubmissionPrice(st *state.StateDB, calldataSize *big.Int) (*big.Int, *big.Int, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetTimeout(st *state.StateDB, ticketId [32]byte) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Keepalive(st *state.StateDB, value *big.Int, ticketId [32]byte) (*big.Int, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Redeem(st *state.StateDB, txId [32]byte) error {
	return errors.New("unimplemented")
}
