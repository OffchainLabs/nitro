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

func (con ArbRetryableTx) Cancel(b burn, caller addr, evm mech, ticketId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) GetBeneficiary(b burn, caller addr, evm mech, ticketId [32]byte) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetKeepalivePrice(b burn, caller addr, evm mech, ticketId [32]byte) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetLifetime(b burn, caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetSubmissionPrice(b burn, caller addr, evm mech, calldataSize huge) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetTimeout(b burn, caller addr, evm mech, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Keepalive(b burn, caller addr, evm mech, value huge, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) Redeem(b burn, caller addr, evm mech, txId [32]byte) error {
	return errors.New("unimplemented")
}
