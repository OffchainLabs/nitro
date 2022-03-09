//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(ctx, mech, bytes32) error
	LifetimeExtended        func(ctx, mech, bytes32, huge) error
	RedeemScheduled         func(ctx, mech, bytes32, bytes32, uint64, uint64, addr) error
	Canceled                func(ctx, mech, bytes32) error
	TicketCreatedGasCost    func(bytes32) (uint64, error)
	LifetimeExtendedGasCost func(bytes32, huge) (uint64, error)
	RedeemScheduledGasCost  func(bytes32, bytes32, uint64, uint64, addr) (uint64, error)
	CanceledGasCost         func(bytes32) (uint64, error)
}

var (
	ErrNotFound               = errors.New("ticketId not found")
	ErrSelfModifyingRetryable = errors.New("retryable cannot modify itself")
)

// Schedule an attempt to redeem the retryable, donating all of the call's gas to the redeem attempt
func (con ArbRetryableTx) Redeem(c ctx, evm mech, ticketId bytes32) (bytes32, error) {
	if c.txProcessor.CurrentRetryable != nil && ticketId == *c.txProcessor.CurrentRetryable {
		return bytes32{}, ErrSelfModifyingRetryable
	}
	retryableState := c.state.RetryableState()
	byteCount, err := retryableState.RetryableSizeBytes(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return hash{}, err
	}
	writeBytes := arbmath.WordsForBytes(byteCount)
	if err := c.Burn(params.SloadGas * writeBytes); err != nil {
		return hash{}, err
	}

	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return hash{}, err
	}
	if retryable == nil {
		return hash{}, ErrNotFound
	}
	nonce, err := retryable.IncrementNumTries()
	if err != nil {
		return hash{}, err
	}

	retryTxInner, err := retryable.MakeTx(
		evm.ChainConfig().ChainID,
		nonce,
		evm.GasPrice,
		0, // will fill this in below
		ticketId,
		c.caller,
	)
	if err != nil {
		return hash{}, err
	}

	// figure out how much gas the event issuance will cost, and reduce the donated gas amount in the event
	//     by that much, so that we'll donate the correct amount of gas
	eventCost, err := con.RedeemScheduledGasCost(hash{}, hash{}, 0, 0, addr{})
	if err != nil {
		return hash{}, err
	}
	// Result is 32 bytes long which is 1 word
	gasCostToReturnResult := params.CopyGas
	gasPoolUpdateCost := storage.StorageReadCost + storage.StorageWriteCost
	futureGasCosts := eventCost + gasCostToReturnResult + gasPoolUpdateCost
	if c.gasLeft < futureGasCosts {
		return hash{}, c.Burn(futureGasCosts) // this will error
	}
	gasToDonate := c.gasLeft - futureGasCosts
	if gasToDonate < params.TxGas {
		return hash{}, errors.New("not enough gas to run redeem attempt")
	}

	// fix up the gas in the retry
	retryTxInner.Gas = gasToDonate

	retryTx := types.NewTx(retryTxInner)
	retryTxHash := retryTx.Hash()

	err = con.RedeemScheduled(c, evm, ticketId, retryTxHash, nonce, gasToDonate, c.caller)
	if err != nil {
		return hash{}, err
	}

	// To prepare for the enqueued retry event, we burn gas here, adding it back to the pool right before retrying.
	// The gas payer for this tx will get a credit for the wei they paid for this gas when retrying.
	// We burn as much gas as we can, leaving only enough to pay for copying out the return data.
	if err := c.Burn(gasToDonate); err != nil {
		return hash{}, err
	}

	// Add the gasToDonate back to the gas pool: the retryable attempt will then consume it.
	// This ensures that the gas pool has enough gas to run the retryable attempt.
	return retryTxHash, c.state.L2PricingState().AddToGasPool(arbmath.SaturatingCast(gasToDonate))
}

// Gets the default lifetime period a retryable has at creation
func (con ArbRetryableTx) GetLifetime(c ctx, evm mech) (huge, error) {
	return big.NewInt(retryables.RetryableLifetimeSeconds), nil
}

// Gets the timestamp for when ticket will expire
func (con ArbRetryableTx) GetTimeout(c ctx, evm mech, ticketId bytes32) (huge, error) {
	retryableState := c.state.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return nil, err
	}
	if retryable == nil {
		return nil, ErrNotFound
	}
	timeout, err := retryable.Timeout()
	if err != nil {
		return nil, err
	}
	return big.NewInt(int64(timeout)), nil
}

// Adds one lifetime period to the ticket's expiry
func (con ArbRetryableTx) Keepalive(c ctx, evm mech, ticketId bytes32) (huge, error) {

	// charge for the expiry update
	retryableState := c.state.RetryableState()
	nbytes, err := retryableState.RetryableSizeBytes(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return nil, err
	}
	if nbytes == 0 {
		return nil, ErrNotFound
	}
	updateCost := arbmath.WordsForBytes(nbytes) * params.SstoreSetGas / 100
	if err := c.Burn(updateCost); err != nil {
		return big.NewInt(0), err
	}

	currentTime := evm.Context.Time.Uint64()
	window := currentTime + retryables.RetryableLifetimeSeconds
	err = retryableState.Keepalive(ticketId, currentTime, window, retryables.RetryableLifetimeSeconds)
	if err != nil {
		return big.NewInt(0), err
	}

	retryable, err := retryableState.OpenRetryable(ticketId, currentTime)
	if err != nil {
		return nil, err
	}
	newTimeout, err := retryable.Timeout()
	if err != nil {
		return nil, err
	}
	err = con.LifetimeExtended(c, evm, ticketId, big.NewInt(int64(newTimeout)))
	if err != nil {
		return nil, err
	}
	return big.NewInt(int64(newTimeout)), nil
}

// Gets the beneficiary of the ticket
func (con ArbRetryableTx) GetBeneficiary(c ctx, evm mech, ticketId bytes32) (addr, error) {
	retryableState := c.state.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return addr{}, err
	}
	if retryable == nil {
		return addr{}, ErrNotFound
	}
	return retryable.Beneficiary()
}

// Cancel the ticket and refund its callvalue to its beneficiary
func (con ArbRetryableTx) Cancel(c ctx, evm mech, ticketId bytes32) error {
	if c.txProcessor.CurrentRetryable != nil && ticketId == *c.txProcessor.CurrentRetryable {
		return ErrSelfModifyingRetryable
	}
	retryableState := c.state.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time.Uint64())
	if err != nil {
		return err
	}
	if retryable == nil {
		return ErrNotFound
	}
	beneficiary, err := retryable.Beneficiary()
	if err != nil {
		return err
	}
	if c.caller != beneficiary {
		return errors.New("only the beneficiary may cancel a retryable")
	}

	// no refunds are given for deleting retryables because they use rented space
	_, err = retryableState.DeleteRetryable(ticketId, evm)
	if err != nil {
		return err
	}
	return con.Canceled(c, evm, ticketId)
}
