// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

type ArbRetryableTx struct {
	Address                 addr
	TicketCreated           func(ctx, mech, bytes32) error
	LifetimeExtended        func(ctx, mech, bytes32, huge) error
	RedeemScheduled         func(ctx, mech, bytes32, bytes32, uint64, uint64, addr, huge, huge) error
	Canceled                func(ctx, mech, bytes32) error
	TicketCreatedGasCost    func(bytes32) (uint64, error)
	LifetimeExtendedGasCost func(bytes32, huge) (uint64, error)
	RedeemScheduledGasCost  func(bytes32, bytes32, uint64, uint64, addr, huge, huge) (uint64, error)
	CanceledGasCost         func(bytes32) (uint64, error)

	// deprecated event
	Redeemed        func(ctx, mech, bytes32) error
	RedeemedGasCost func(bytes32) (uint64, error)

	NoTicketWithIDError func() error
	NotCallableError    func() error
}

var ErrSelfModifyingRetryable = errors.New("retryable cannot modify itself")

func (con ArbRetryableTx) oldNotFoundError(c ctx) error {
	if c.State.ArbOSVersion() >= 3 {
		return con.NoTicketWithIDError()
	} else {
		return errors.New("ticketId not found")
	}
}

// Redeem schedules an attempt to redeem the retryable, donating all of the call's gas to the redeem attempt
func (con ArbRetryableTx) Redeem(c ctx, evm mech, ticketId bytes32) (bytes32, error) {
	if c.txProcessor.CurrentRetryable != nil && ticketId == *c.txProcessor.CurrentRetryable {
		return bytes32{}, ErrSelfModifyingRetryable
	}
	retryableState := c.State.RetryableState()
	writeWords, err := retryableState.RetryableSizeWords(ticketId, evm.Context.Time)
	if err != nil {
		return hash{}, err
	}
	if err := c.Burn(params.SloadGas * writeWords); err != nil {
		return hash{}, err
	}

	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time)
	if err != nil {
		return hash{}, err
	}
	if retryable == nil {
		return hash{}, con.oldNotFoundError(c)
	}
	nextNonce, err := retryable.IncrementNumTries()
	if err != nil {
		return hash{}, err
	}
	nonce := nextNonce - 1

	maxRefund := new(big.Int).Exp(common.Big2, common.Big256, nil)
	maxRefund.Sub(maxRefund, common.Big1)
	retryTxInner, err := retryable.MakeTx(
		evm.ChainConfig().ChainID,
		nonce,
		evm.Context.BaseFee,
		0, // will fill this in below
		ticketId,
		c.caller,
		maxRefund,
		common.Big0,
	)
	if err != nil {
		return hash{}, err
	}
	// figure out how much gas the event issuance will cost, and reduce the donated gas amount in the event
	//     by that much, so that we'll donate the correct amount of gas
	eventCost, err := con.RedeemScheduledGasCost(hash{}, hash{}, 0, 0, addr{}, common.Big0, common.Big0)
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

	err = con.RedeemScheduled(c, evm, ticketId, retryTxHash, nonce, gasToDonate, c.caller, maxRefund, common.Big0)
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
	return retryTxHash, c.State.L2PricingState().AddToGasPool(arbmath.SaturatingCast(gasToDonate))
}

func checkValidArchivedAndRedeemable(
	c ctx, retryFrom addr, l1BaseFee *big.Int,
	chainId *big.Int, ticketId bytes32,
	requestId bytes32, l1BaseFee, deposit, callValue, gasFeeCap huge,
	gasLimit uint64, maxSubmissionFee huge,
	feeRefundAddress, beneficiary, retryTo addr,
	retryData []byte,
	rootHash common.Hash,
	leafIndex uint64,
	proof []common.Hash,
) error {
	txHash := types.NewTx(&types.ArbitrumSubmitRetryableTx{
		ChainId:          chainId,
		RequestId:        common.BytesToHash(requestId),
		From:             retryFrom,
		L1BaseFee:        l1BaseFee,
		DepositValue:     deposit,
		GasFeeCap:        gasFeeCap,
		Gas:              gasLimit,
		RetryTo:          retryTo,
		RetryValue:       callValue,
		Beneficiary:      beneficiary,
		MaxSubmissionFee: maxSubmissionFee,
		FeeRefundAddr:    feeRefundAddress,
		RetryData:        retryData,
	}).Hash().Bytes()
	if !bytes.Equal(ticketId, txHash) {
		// TODO(magic) err
		return errors.New("TODO")
	}
	retryableState := c.State.RetryableState()
	archiveRoot, err := retryableState.Archive.Root()
	if err != nil {
		return bytes32{}, err
	}
	if !bytes.Equal(rootHash.Bytes(), archiveRoot.Bytes()) {
		// TODO(magic) err
		return errors.New("TODO")
	}
	merkleProof := merkletree.MerkleProof{
		RootHash:  rootHash,
		LeafHash:  common.BytesToHash(crypto.Keccak256(ticketId)),
		LeafIndex: leafIndex,
		Proof:     proof,
	}
	if !merkleProof.IsCorrect() {
		// TODO(magic) err
		return errors.New("TODO")
	}
	isNonRedeemable, err := retryableState.NonRedeemableArchived.IsMember(leafIndex)
	if isNonRedeemable {
		// TODO(magic) err
		return errors.New("TODO")
	}
	return nil
}

func (con ArbRetryableTx) RedeemArchived(c ctx, evm mech,
	ticketId bytes32,
	requestId bytes32, l1BaseFee, deposit, callValue, gasFeeCap huge,
	gasLimit uint64, maxSubmissionFee huge,
	retryFrom, feeRefundAddress, beneficiary, retryTo addr,
	retryData []byte,
	rootHash common.Hash,
	leafIndex uint64,
	proof []common.Hash,
) (bytes32, error) {
	// TODO(magic) verify gas accounting
	// TODO(magic) verify addresses

	// TODO(magic) is it ok to check ticketId before verifying if it's valid?
	if c.txProcessor.CurrentRetryable != nil && ticketId == *c.txProcessor.CurrentRetryable {
		return bytes32{}, ErrSelfModifyingRetryable
	}
	chainId := evm.ChainConfig.ChainID
	if err := checkValidArchivedAndRedeemable(
		c, retryFrom, l1BaseFee, chainId, ticketId, requestId, l1BaseFee, deposit, callValue, gasFeeCap, gasLimit,
		maxSubmissionFee, feeRefundAddress, beneficiary, retryTo, retryData, rootHash, leafIndex, proof); err != nil {
		return bytes32{}, err
	}
	maxRefund := new(big.Int).Exp(common.Big2, common.Big256, nil)
	maxRefund.Sub(maxRefund, common.Big1)

	// TODO(magic) fix nonce collision with previous attempts to redeem the retryable before its expiry
	nextNonce, err := retrayableState.IncrementNumArchiveTries()
	if err != nil {
		return hash{}, err
	}
	nonce := nextNonce - 1
	// figure out how much gas the event issuance will cost, and reduce the donated gas amount in the event
	//     by that much, so that we'll donate the correct amount of gas
	eventCost, err := con.RedeemArchivedScheduledGasCost(hash{}, hash{}, 0, 0, addr{}, common.Big0, common.Big0, addr{}, addr{}, common.Big0, byte[len(callData)]{})
	if err != nil {
		return hash{}, err
	}
	// Result is 32 bytes long which is 1 word
	gasCostToReturnResult := params.CopyGas
	gasPoolUpdateCost := storage.StorageReadCost + storage.StorageWriteCost
	futureGasCosts := eventCost + gasCostToReturnResult + gasPoolUpdateCost
	if err != nil {
		return hash{}, err
	}
	retryableState := c.State.RetryableState()
	// account for marking the retryable as no longer redeemable
	futureGasCosts += retrayableState.NonRedeemableArchived.AddMaxGasCost()
	if c.gasLeft < futureGasCosts {
		return hash{}, c.Burn(futureGasCosts) // this will error
	}
	gasToDonate := c.gasLeft - futureGasCosts
	if gasToDonate < params.TxGas {
		return hash{}, errors.New("not enough gas to run redeem attempt")
	}
	retryTxInner := &types.ArbitrumRetryTx{
		ChainId:             chainId,
		Nonce:               nonce,
		From:                retryFrom,
		GasFeeCap:           evm.Context.BaseFee,
		Gas:                 gasToDonate,
		To:                  retryTo,
		Value:               callValue,
		Data:                callData,
		TicketId:            ticketId,
		RefundTo:            feeRefundAddress,
		MaxRefund:           maxRefund,
		SubmissionFeeRefund: common.Big0,
	}
	retryTx := types.NewTx(retryTxInner)
	retryTxHash := retryTx.Hash()

	// TODO(magic) do we need retryTx hash if we are passing all the data?
	if err = con.RedeemArchivedScheduled(c, evm, ticketId, retryTxHash, nonce, gasToDonate, c.caller, maxRefund, common.Big0, retryFrom, callValue, callData); err != nil {
		return hash{}, err
	}
	if _, err = retryableState.NonRedeemableArchived.Add(leafIndex); err != nil {
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
	return retryTxHash, c.State.L2PricingState().AddToGasPool(arbmath.SaturatingCast(gasToDonate))
}

// GetLifetime gets the default lifetime period a retryable has at creation
func (con ArbRetryableTx) GetLifetime(c ctx, evm mech) (huge, error) {
	return big.NewInt(retryables.RetryableLifetimeSeconds), nil
}

// GetTimeout gets the timestamp for when ticket will expire
func (con ArbRetryableTx) GetTimeout(c ctx, evm mech, ticketId bytes32) (huge, error) {
	retryableState := c.State.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time)
	if err != nil {
		return nil, err
	}
	if retryable == nil {
		return nil, con.NoTicketWithIDError()
	}
	timeout, err := retryable.CalculateTimeout()
	if err != nil {
		return nil, err
	}
	return big.NewInt(int64(timeout)), nil
}

// Keepalive adds one lifetime period to the ticket's expiry
func (con ArbRetryableTx) Keepalive(c ctx, evm mech, ticketId bytes32) (huge, error) {

	// charge for the expiry update
	retryableState := c.State.RetryableState()
	nwords, err := retryableState.RetryableSizeWords(ticketId, evm.Context.Time)
	if err != nil {
		return nil, err
	}
	if nwords == 0 {
		return nil, con.oldNotFoundError(c)
	}
	updateCost := nwords * params.SstoreSetGas / 100
	if err := c.Burn(updateCost); err != nil {
		return big.NewInt(0), err
	}

	currentTime := evm.Context.Time
	window := currentTime + retryables.RetryableLifetimeSeconds
	newTimeout, err := retryableState.Keepalive(ticketId, currentTime, window, retryables.RetryableLifetimeSeconds)
	if err != nil {
		return big.NewInt(0), err
	}

	err = con.LifetimeExtended(c, evm, ticketId, big.NewInt(int64(newTimeout)))
	return big.NewInt(int64(newTimeout)), err
}

// GetBeneficiary gets the beneficiary of the ticket
func (con ArbRetryableTx) GetBeneficiary(c ctx, evm mech, ticketId bytes32) (addr, error) {
	retryableState := c.State.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time)
	if err != nil {
		return addr{}, err
	}
	if retryable == nil {
		return addr{}, con.oldNotFoundError(c)
	}
	return retryable.Beneficiary()
}

// Cancel the ticket and refund its callvalue to its beneficiary
func (con ArbRetryableTx) Cancel(c ctx, evm mech, ticketId bytes32) error {
	if c.txProcessor.CurrentRetryable != nil && ticketId == *c.txProcessor.CurrentRetryable {
		return ErrSelfModifyingRetryable
	}
	retryableState := c.State.RetryableState()
	retryable, err := retryableState.OpenRetryable(ticketId, evm.Context.Time)
	if err != nil {
		return err
	}
	if retryable == nil {
		return con.oldNotFoundError(c)
	}
	beneficiary, err := retryable.Beneficiary()
	if err != nil {
		return err
	}
	if c.caller != beneficiary {
		return errors.New("only the beneficiary may cancel a retryable")
	}

	// no refunds are given for deleting retryables because they use rented space
	_, err = retryableState.DeleteRetryable(ticketId, evm, util.TracingDuringEVM)
	if err != nil {
		return err
	}
	return con.Canceled(c, evm, ticketId)
}

func (con ArbRetryableTx) CancelArchived(c ctx, evm mech,
	ticketId bytes32,
	requestId bytes32, l1BaseFee, deposit, callValue, gasFeeCap huge,
	gasLimit uint64, maxSubmissionFee huge,
	retryFrom, feeRefundAddress, beneficiary, retryTo addr,
	retryData []byte,
	rootHash common.Hash,
	leafIndex uint64,
	proof []common.Hash,
) error {
	chainId := evm.ChainConfig.ChainID
	if err := checkValidArchivedAndRedeemable(
		c, retryFrom, l1BaseFee, chainId, ticketId, requestId, l1BaseFee, deposit, callValue, gasFeeCap, gasLimit,
		maxSubmissionFee, feeRefundAddress, beneficiary, retryTo, retryData, rootHash, leafIndex, proof); err != nil {
		return err
	}
	if c.caller != beneficiary {
		return errors.New("only the beneficiary may cancel a retryable")
	}
	retrayableState := c.State.RetryableState()
	if _, err := retrayableState.NonRedeemableArchived.Add(leafIndex); err != nil {
		return err
	}
	retryables.MoveFundsLeftInEscrowToBeneficiary(ticketId, beneficiary, evm, scenario)
	return con.Canceled(c, evm, ticketId)
}

func (con ArbRetryableTx) GetCurrentRedeemer(c ctx, evm mech) (common.Address, error) {
	if c.txProcessor.CurrentRefundTo != nil {
		return *c.txProcessor.CurrentRefundTo, nil
	} else {
		return common.Address{}, nil
	}
}

func (con ArbRetryableTx) SubmitRetryable(
	c ctx, evm mech, requestId bytes32, l1BaseFee, deposit, callValue, gasFeeCap huge,
	gasLimit uint64, maxSubmissionFee huge,
	feeRefundAddress, beneficiary, retryTo addr,
	retryData []byte,
) error {
	return con.NotCallableError()
}
