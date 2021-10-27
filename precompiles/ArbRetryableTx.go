//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(mech, [32]byte)
	LifetimeExtended        func(mech, [32]byte, huge)
	Redeemed                func(mech, [32]byte)
	RedeemScheduled         func(mech, [32]byte, [32]byte, huge, huge)
	Canceled                func(mech, [32]byte)
	TicketCreatedGasCost    func([32]byte) uint64
	LifetimeExtendedGasCost func([32]byte, huge) uint64
	RedeemedGasCost         func([32]byte) uint64
	RedeemScheduledGasCost  func([32]byte, [32]byte, huge, huge) uint64
	CanceledGasCost         func([32]byte) uint64
}

const RetryableLifetimeSeconds = 7 * 24 * 60 * 60 // one week

var (
	NotFoundError     = errors.New("ticketId not found")
	UnauthorizedError = errors.New("unauthorized caller")
)

func (con ArbRetryableTx) Cancel(c ctx, caller addr, evm mech, ticketId [32]byte) error {
	if err := c.burn(con.CanceledGasCost(ticketId)); err != nil {
		return err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	retryable := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return NotFoundError
	}
	if caller != retryable.Beneficiary() {
		return UnauthorizedError
	}
	retryableState.DeleteRetryable(ticketId)
	con.Canceled(evm, ticketId)
	return nil
}

func (con ArbRetryableTx) GetBeneficiary(c ctx, caller addr, evm mech, ticketId [32]byte) (addr, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return addr{}, err
	}
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Address{}, NotFoundError
	}
	return retryable.Beneficiary(), nil
}

func (con ArbRetryableTx) GetKeepaliveGas(c ctx, caller addr, evm mech, ticketId [32]byte) (huge, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	nbytes := arbos.OpenArbosState(evm.StateDB).RetryableState().RetryableSizeBytes(ticketId, evm.Context.Time.Uint64())
	if nbytes == 0 {
		return nil, NotFoundError
	}
	return big.NewInt(int64(util.WordsForBytes(nbytes) * params.SstoreSetGas / 100)), nil
}

func (con ArbRetryableTx) GetLifetime(c ctx, caller addr, evm mech) (huge, error) {
	if err := c.burn(1); err != nil {
		return nil, err
	}
	return big.NewInt(RetryableLifetimeSeconds), nil
}

func (con ArbRetryableTx) GetTimeout(c ctx, caller addr, evm mech, ticketId [32]byte) (huge, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return nil, NotFoundError
	}
	return big.NewInt(int64(retryable.Timeout())), nil
}

func (con ArbRetryableTx) Keepalive(c ctx, caller addr, evm mech, value huge, ticketId [32]byte) (huge, error) {
	if err := c.burn(3*params.SloadGas + 2*params.SstoreSetGas + con.LifetimeExtendedGasCost(ticketId, nil)); err != nil {
		return nil, err
	}
	currentTime := evm.Context.Time.Uint64()
	rs := arbos.OpenArbosState(evm.StateDB).RetryableState()
	success := rs.Keepalive(ticketId, currentTime, currentTime+RetryableLifetimeSeconds, RetryableLifetimeSeconds)
	if !success {
		return nil, NotFoundError
	}
	newTimeout := rs.OpenRetryable(ticketId, currentTime).Timeout()
	con.LifetimeExtended(evm, ticketId, big.NewInt(int64(newTimeout)))
	return big.NewInt(int64(newTimeout)), nil
}

const MockRedeemGasAvailableBUGBUGBUG uint64 = 1000000

func (con ArbRetryableTx) Redeem(c ctx, caller addr, evm mech, txId [32]byte) ([32]byte, error) {
	if err := c.burn(5 * params.SloadGas + params.SstoreSetGas + con.RedeemScheduledGasCost(txId, txId, nil, nil)); err != nil {
		return common.Hash{}, err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	if err := c.burn(params.SloadGas * util.WordsForBytes(retryableState.RetryableSizeBytes(txId, evm.Context.Time.Uint64()))); err != nil {
		return common.Hash{}, err
	}
	retryable := retryableState.OpenRetryable(txId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Hash{}, NotFoundError
	}
	sequenceNum := retryable.IncrementNumTries()
	donatedGas := MockRedeemGasAvailableBUGBUGBUG
	redeemTxId := crypto.Keccak256Hash(txId[:], common.BigToHash(sequenceNum).Bytes())
	con.RedeemScheduled(evm, txId, redeemTxId, sequenceNum, big.NewInt(int64(donatedGas)))
	return redeemTxId, nil
}
