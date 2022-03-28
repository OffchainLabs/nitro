//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewArbitrumSubmitRetryableTx(
	chainId *big.Int,
	requestId common.Hash,
	from common.Address,
	l1BaseFee,
	depositValue,
	maxFeePerGas *big.Int,
	gasLimit uint64,
	retryTo *common.Address,
	callvalue *big.Int,
	callvalueRefundAddress common.Address,
	maxSubmissionFee *big.Int,
	feeRefundAddress common.Address,
	retryData []byte,
) (*types.ArbitrumSubmitRetryableTx, error) {
	tx := &types.ArbitrumSubmitRetryableTx{
		ChainId:          chainId,
		RequestId:        requestId,
		From:             from,
		L1BaseFee:        l1BaseFee,
		DepositValue:     depositValue,
		GasFeeCap:        maxFeePerGas,
		Gas:              gasLimit,
		RetryTo:          retryTo,
		Value:            callvalue,
		Beneficiary:      callvalueRefundAddress,
		MaxSubmissionFee: maxSubmissionFee,
		FeeRefundAddr:    feeRefundAddress,
		RetryData:        retryData,
	}
	toToEncode := common.Address{}
	if tx.RetryTo != nil {
		toToEncode = *tx.RetryTo
	}
	data, err := PackArbRetryableTxSubmitRetryable(
		tx.RequestId,
		tx.L1BaseFee,
		tx.DepositValue,
		tx.Value,
		tx.GasFeeCap,
		tx.Gas,
		tx.MaxSubmissionFee,
		tx.FeeRefundAddr,
		tx.Beneficiary,
		toToEncode,
		tx.RetryData,
	)
	tx.Data = data
	if err != nil {
		return tx, fmt.Errorf("Failed to abi-encode submission data %w", err)
	}
	return tx, nil
}
