//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/util"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(mech, [32]byte)
	LifetimeExtended        func(mech, [32]byte, huge)
	RedeemScheduled         func(mech, [32]byte, [32]byte, uint64, uint64, addr)
	Redeemed                func(mech, [32]byte)
	Canceled                func(mech, [32]byte)
	TicketCreatedGasCost    func([32]byte) uint64
	LifetimeExtendedGasCost func([32]byte, huge) uint64
	RedeemScheduledGasCost  func([32]byte, [32]byte, uint64, uint64, addr) uint64
	RedeemedGasCost         func([32]byte) uint64
	CanceledGasCost         func([32]byte) uint64
}

var NotFoundError = errors.New("ticketId not found")

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
		return errors.New("only the beneficiary may cancel a retryable")
	}

	// no refunds are given for deleting retryables because they use rented space
	retryableState.DeleteRetryable(ticketId)
	con.Canceled(evm, ticketId)
	return nil
}

func (con ArbRetryableTx) GetBeneficiary(c ctx, evm mech, ticketId [32]byte) (addr, error) {
	if err := c.burn(2 * params.SloadGas); err != nil {
		return addr{}, err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	retryable := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Address{}, NotFoundError
	}
	return retryable.Beneficiary(), nil
}

func (con ArbRetryableTx) GetLifetime(c ctx, evm mech) (huge, error) {
	// there's no need to burn gas for something this cheap
	return big.NewInt(retryables.RetryableLifetimeSeconds), nil
}

func (con ArbRetryableTx) GetTimeout(c ctx, evm mech, ticketId [32]byte) (huge, error) {
	if err := c.burn(3 * params.SloadGas); err != nil {
		return big.NewInt(0), err
	}
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	retryable := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return big.NewInt(0), NotFoundError
	}
	return big.NewInt(int64(retryable.Timeout())), nil
}

func (con ArbRetryableTx) Keepalive(c ctx, evm mech, ticketId [32]byte) (huge, error) {

	// charge for the check & event
	eventCost := con.LifetimeExtendedGasCost(ticketId, big.NewInt(0))
	if err := c.burn(3*params.SloadGas + 2*params.SstoreSetGas + eventCost); err != nil {
		return big.NewInt(0), err
	}

	// charge for the expiry update
	updateCost, err := con.GetKeepaliveGas(c, evm, ticketId)
	if err != nil {
		return big.NewInt(0), err
	}
	if err = c.burn(updateCost.Uint64()); err != nil {
		return big.NewInt(0), err
	}

	currentTime := evm.Context.Time.Uint64()
	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	window := currentTime + retryables.RetryableLifetimeSeconds
	err = retryableState.Keepalive(ticketId, currentTime, window, retryables.RetryableLifetimeSeconds)
	if err != nil {
		return big.NewInt(0), err
	}

	newTimeout := retryableState.OpenRetryable(ticketId, currentTime).Timeout()
	con.LifetimeExtended(evm, ticketId, big.NewInt(int64(newTimeout)))
	return big.NewInt(int64(newTimeout)), nil
}

func (con ArbRetryableTx) Redeem(c ctx, evm mech, ticketId [32]byte) ([32]byte, error) {

	eventCost := con.RedeemScheduledGasCost(ticketId, ticketId, 0, 0, c.caller)
	if err := c.burn(5*params.SloadGas + params.SstoreSetGas + eventCost); err != nil {
		return common.Hash{}, err
	}

	retryableState := arbos.OpenArbosState(evm.StateDB).RetryableState()
	writeBytes := util.WordsForBytes(retryableState.RetryableSizeBytes(ticketId, evm.Context.Time.Uint64()))
	if err := c.burn(params.SloadGas * writeBytes); err != nil {
		return common.Hash{}, err
	}

	retryable := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if retryable == nil {
		return common.Hash{}, NotFoundError
	}
	sequenceNum := retryable.IncrementNumTries()
	redeemTxId := retryables.TxIdForRedeemAttempt(ticketId, sequenceNum)
	con.RedeemScheduled(evm, ticketId, redeemTxId, sequenceNum, c.gasLeft, c.caller)

	// now donate all of the remaining gas to the retry
	// to do this, we burn the gas here, but add it back into the gas pool just before the retry runs
	// the gas payer for this transaction will get a credit for the wei they paid for this gas, when the retry occurs
	if err := c.burn(c.gasLeft); err != nil {
		return common.Hash{}, err
	}

	return redeemTxId, nil
}
