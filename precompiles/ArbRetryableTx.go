//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type ArbRetryableTx struct {
	Address          addr
	TicketCreated    func(mech, [32]byte)
	LifetimeExtended func(mech, [32]byte, huge)
	RedeemScheduled  func(mech, [32]byte, [32]byte, huge, huge)
	Redeemed         func(mech, [32]byte)
	Canceled         func(mech, [32]byte)
}

const RetryableLifetimeSeconds = 7*24*60*60   // one week

var (
	NotFoundError = errors.New("ticketId not found")
)

func (con ArbRetryableTx) Cancel(caller addr, evm mech, ticketId [32]byte) error {
	arbos.OpenArbosState(evm.StateDB).RetryableState().DeleteRetryable(ticketId)
	return nil
}

func (con ArbRetryableTx) CancelGasCost(ticketId [32]byte) uint64 {
	return 0
}

func (con ArbRetryableTx) GetBeneficiary(caller addr, evm mech, ticketId [32]byte) (addr, error) {
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Address{}, NotFoundError
	}
	return retryable.Beneficiary(), nil
}

func (con ArbRetryableTx) GetBeneficiaryGasCost(ticketId [32]byte) uint64 {
	return 3 * params.SloadGas
}

func (con ArbRetryableTx) GetKeepaliveGas(caller addr, evm mech, ticketId [32]byte) (huge, error) {
	nbytes := arbos.OpenArbosState(evm.StateDB).RetryableState().RetryableSizeBytes(ticketId)
	if nbytes == 0 {
		return nil, NotFoundError
	}
	return big.NewInt(int64(util.WordsForBytes(nbytes) * params.SstoreSetGas / 100)), nil
}

func (con ArbRetryableTx) GetKeepaliveGasGasCost(ticketId [32]byte) uint64 {
	return 3 * params.SloadGas
}

func (con ArbRetryableTx) GetLifetime(caller addr, evm mech) (huge, error) {
	return big.NewInt(RetryableLifetimeSeconds), nil
}

func (con ArbRetryableTx) GetLifetimeGasCost() uint64 {
	return 0
}

func (con ArbRetryableTx) GetTimeout(caller addr, evm mech, ticketId [32]byte) (huge, error) {
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return nil, NotFoundError
	}
	return retryable.Timeout(), nil
}

func (con ArbRetryableTx) GetTimeoutGasCost(ticketId [32]byte) uint64 {
	return 3 * params.SloadGas
}

func (con ArbRetryableTx) Keepalive(caller addr, evm mech, value huge, ticketId [32]byte) (huge, error) {
	currentTime := evm.Context.Time.Uint64()
	rs := arbos.OpenArbosState(evm.StateDB).RetryableState()
	success := rs.Keepalive(ticketId, currentTime, currentTime+RetryableLifetimeSeconds, RetryableLifetimeSeconds)
	if !success {
		return nil, NotFoundError
	}
	return rs.OpenRetryable(ticketId, currentTime).Timeout(), nil
}

func (con ArbRetryableTx) KeepaliveGasCost(ticketId [32]byte) uint64 {
	return 3 * params.SloadGas + 2 * params.SstoreSetGas //TODO: add big.NewInt(int64(util.WordsForBytes(nbytes) * params.SstoreSetGas / 100))
}

func (con ArbRetryableTx) Redeem(caller addr, evm mech, txId [32]byte) ([32]byte, error) {
	return [32]byte{}, errors.New("unimplemented")
}

func (con ArbRetryableTx) RedeemGasCost(txId [32]byte) uint64 {
	return 0
}
