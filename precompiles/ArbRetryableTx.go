//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(mech, [32]byte)
	LifetimeExtended        func(mech, [32]byte, huge)
	Redeemed                func(mech, [32]byte)
	Canceled                func(mech, [32]byte)
	TicketCreatedGasCost    func([32]byte) uint64
	LifetimeExtendedGasCost func([32]byte, huge) uint64
	RedeemedGasCost         func([32]byte) uint64
	CanceledGasCost         func([32]byte) uint64
}

func (con ArbRetryableTx) Cancel(c ctx, evm mech, ticketId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) GetBeneficiary(c ctx, evm mech, ticketId [32]byte) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetKeepalivePrice(c ctx, evm mech, ticketId [32]byte) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetLifetime(c ctx, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetSubmissionPrice(c ctx, evm mech, calldataSize huge) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetTimeout(c ctx, evm mech, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Keepalive(c ctx, evm mech, value huge, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Redeem(c ctx, evm mech, txId [32]byte) error {
	return errors.New("unimplemented")
}
