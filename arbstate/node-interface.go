//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
)

// To avoid creating new RPC methods for client-side tooling, nitro Geth's InterceptRPCMessage() hook provides
// an opportunity to swap out the message its handling before deriving a transaction from it.
//
// This function handles messages sent to 0xc8 and uses NodeInterface.sol to determine what to do. No contract
// actually exists at 0xc8, but the abi methods allow the incoming message's calldata to specify the arguments.
//
func ApplyNodeInterface(msg types.Message, nodeInterface abi.ABI) (types.Message, error) {

	estimateMethod := nodeInterface.Methods["estimateRetryableTicket"]

	calldata := msg.Data()
	if len(calldata) < 4 {
		return msg, errors.New("calldata for NodeInterface.sol is too short")
	}

	if bytes.Equal(estimateMethod.ID, calldata[:4]) {
		inputs, err := estimateMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, err
		}
		sender, _ := inputs[0].(common.Address)
		deposit, _ := inputs[1].(*big.Int)
		destAddr, _ := inputs[2].(common.Address)
		l2CallValue, _ := inputs[3].(*big.Int)
		excessFeeRefundAddress, _ := inputs[4].(common.Address)
		callValueRefundAddress, _ := inputs[5].(common.Address)
		data, _ := inputs[6].([]byte)

		var to *common.Address
		if destAddr != (common.Address{}) {
			to = &destAddr
		}

		tx := types.NewTx(&types.ArbitrumSubmitRetryableTx{
			ChainId:       nil,
			RequestId:     common.Hash{},
			From:          sender,
			DepositValue:  deposit,
			GasPrice:      math.MaxBig256,
			Gas:           msg.Gas(),
			To:            to,
			Value:         l2CallValue,
			Beneficiary:   callValueRefundAddress,
			FeeRefundAddr: excessFeeRefundAddress,
			Data:          data,
		})

		// ArbitrumSubmitRetryableTx is unsigned so the following won't panic
		return tx.AsMessage(types.NewArbitrumSigner(nil), nil)
	}

	return msg, errors.New("method does not exist in NodeInterface.sol")
}
