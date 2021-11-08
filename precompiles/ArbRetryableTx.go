//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/arbos/util"
	mathutil "github.com/offchainlabs/arbstate/util"
	"math/big"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(mech, [32]byte)
	LifetimeExtended        func(mech, [32]byte, huge)
	Redeemed                func(mech, [32]byte)
	RedeemScheduled         func(mech, [32]byte, [32]byte, huge, huge, addr)
	Canceled                func(mech, [32]byte)
	TicketCreatedGasCost    func([32]byte) uint64
	LifetimeExtendedGasCost func([32]byte, huge) uint64
	RedeemedGasCost         func([32]byte) uint64
	RedeemScheduledGasCost  func([32]byte, [32]byte, huge, huge, addr) uint64
	CanceledGasCost         func([32]byte) uint64
}

const RetryableLifetimeSeconds = 7 * 24 * 60 * 60 // one week

var (
	NotFoundError = errors.New("ticketId not found")
)

func (con ArbRetryableTx) Cancel(c ctx, evm mech, ticketId [32]byte) error {
	if err := c.burn(con.CanceledGasCost(ticketId)); err != nil {
		return err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	retryable := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return NotFoundError
	}
	if c.caller != retryable.Beneficiary() {
		return UnauthorizedError
	}
	retryableState.DeleteRetryable(ticketId)
	con.Canceled(evm, ticketId)
	return nil
}

func (con ArbRetryableTx) GetBeneficiary(c ctx, evm mech, ticketId [32]byte) (addr, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return addr{}, err
	}
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Address{}, NotFoundError
	}
	return retryable.Beneficiary(), nil
}

func (con ArbRetryableTx) GetKeepaliveGas(c ctx, evm mech, ticketId [32]byte) (huge, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return nil, err
	}
	nbytes := arbos.OpenArbosState(evm.StateDB).RetryableState().RetryableSizeBytes(ticketId, evm.Context.Time.Uint64())
	if nbytes == 0 {
		return nil, NotFoundError
	}
	return big.NewInt(int64(util.WordsForBytes(nbytes) * params.SstoreSetGas / 100)), nil
}

func (con ArbRetryableTx) GetLifetime(c ctx, evm mech) (huge, error) {
	if err := c.burn(1); err != nil {
		return nil, err
	}
	return big.NewInt(RetryableLifetimeSeconds), nil
}

func (con ArbRetryableTx) GetTimeout(c ctx, evm mech, ticketId [32]byte) (huge, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return big.NewInt(0), err
	}
	retryable := arbos.OpenArbosState(evm.StateDB).RetryableState().OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return big.NewInt(0), NotFoundError
	}
	return big.NewInt(int64(retryable.Timeout())), nil
}

func (con ArbRetryableTx) Keepalive(c ctx, evm mech, value huge, ticketId [32]byte) (huge, error) {
	if err := c.burn(3*params.SloadGas + 2*params.SstoreSetGas + con.LifetimeExtendedGasCost(ticketId, mathutil.BigZero)); err != nil {
		return big.NewInt(0), err
	}
	currentTime := evm.Context.Time.Uint64()
	rs := arbos.OpenArbosState(evm.StateDB).RetryableState()
	success := rs.Keepalive(ticketId, currentTime, currentTime+RetryableLifetimeSeconds, RetryableLifetimeSeconds)
	if !success {
		return big.NewInt(0), NotFoundError
	}
	newTimeout := rs.OpenRetryable(ticketId, currentTime).Timeout()
	con.LifetimeExtended(evm, ticketId, big.NewInt(int64(newTimeout)))
	return big.NewInt(int64(newTimeout)), nil
}

func (con ArbRetryableTx) Redeem(c ctx, evm mech, txId [32]byte) ([32]byte, error) {
	if err := c.burn(5*params.SloadGas + params.SstoreSetGas + con.RedeemScheduledGasCost(txId, txId, mathutil.BigZero, mathutil.BigZero, c.caller)); err != nil {
		return common.Hash{}, err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	if err := c.burn(params.SloadGas * util.WordsForBytes(retryableState.RetryableSizeBytes(txId, evm.Context.Time.Uint64()))); err != nil {
		return common.Hash{}, err
	}
	retryable := retryableState.OpenRetryable(txId, evm.Context.Time.Uint64())
	if retryable == nil {
		fmt.Println("Retryable ", txId, " doesn't exist")
		return common.Hash{}, NotFoundError
	}
	sequenceNum := retryable.IncrementNumTries()
	redeemTxId := retryables.TxIdForRedeemAttempt(txId, sequenceNum)
	con.RedeemScheduled(evm, txId, redeemTxId, sequenceNum, big.NewInt(int64(c.gasLeft)), c.caller)

	// now donate all of the remaining gas to the retry
	// to do this, we burn the gas here, but add it back into the gas pool just before the retry runs
	// the gas payer for this transaction will get a credit for the wei they paid for this gas, when the retry occurs
	if err := c.burn(c.gasLeft); err != nil {
		return common.Hash{}, err
	}

	return redeemTxId, nil
}
