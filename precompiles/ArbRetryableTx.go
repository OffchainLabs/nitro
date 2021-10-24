//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbRetryableTx struct {
	Address          addr
	TicketCreated    func(mech, [32]byte)
	LifetimeExtended func(mech, [32]byte, huge)
	Redeemed         func(mech, [32]byte)
	Canceled         func(mech, [32]byte)
}

func (con ArbRetryableTx) Cancel(caller addr, evm mech, ticketId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) CancelGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetBeneficiary(caller addr, evm mech, ticketId [32]byte) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetBeneficiaryGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetKeepalivePrice(caller addr, evm mech, ticketId [32]byte) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetKeepalivePriceGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetLifetime(caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetLifetimeGasCost() uint64 {
	return 0
}

func (con ArbRetryableTx) GetSubmissionPrice(caller addr, evm mech, calldataSize huge) (huge, huge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetSubmissionPriceGasCost(calldataSize huge) uint64 {
	return 0
}

func (con ArbRetryableTx) GetTimeout(caller addr, evm mech, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) GetTimeoutGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) Keepalive(caller addr, evm mech, value huge, ticketId [32]byte) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbRetryableTx) KeepaliveGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) Redeem(caller addr, evm mech, txId [32]byte) error {
	return errors.New("unimplemented")
}

func (con ArbRetryableTx) RedeemGasCost(txId [32]byte) uint64 {
	return 0
}
