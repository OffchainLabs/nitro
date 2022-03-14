//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"bytes"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/util/arbmath"
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
		to, _ := inputs[2].(common.Address)
		l2CallValue, _ := inputs[3].(*big.Int)
		excessFeeRefundAddress, _ := inputs[4].(common.Address)
		callValueRefundAddress, _ := inputs[5].(common.Address)
		data, _ := inputs[6].([]byte)

		var pTo *common.Address
		if to != (common.Address{}) {
			pTo = &to
		}

		tx := types.NewTx(&types.ArbitrumSubmitRetryableTx{
			ChainId:       nil,
			RequestId:     common.Hash{},
			From:          util.RemapL1Address(sender),
			DepositValue:  deposit,
			GasFeeCap:     msg.GasPrice(),
			Gas:           msg.Gas(),
			To:            pTo,
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

func init() {

	nodeInterface, err := abi.JSON(strings.NewReader(node_interfacegen.NodeInterfaceABI))
	if err != nil {
		panic(err)
	}
	core.InterceptRPCMessage = func(msg types.Message) (types.Message, error) {
		to := msg.To()
		if to == nil || *to != common.HexToAddress("0xc8") {
			return msg, nil
		}
		return ApplyNodeInterface(msg, nodeInterface)
	}

	core.InterceptRPCGasCap = func(gascap *uint64, msg types.Message, header *types.Header, statedb *state.StateDB) {
		arbosVersion := arbosState.ArbOSVersion(statedb)
		if arbosVersion == 0 {
			// ArbOS hasn't been installed, so use the vanilla gas cap
			return
		}
		state, err := arbosState.OpenSystemArbosState(statedb, true)
		if err != nil {
			log.Error("ArbOS is not initialized", "err", err)
			return
		}
		poster, _ := state.L1PricingState().ReimbursableAggregatorForSender(msg.From())
		if poster == nil || header.BaseFee.Sign() == 0 {
			// if gas is free or there's no reimbursable poster, the user won't pay for L1 data costs
			return
		}
		posterCost, _ := state.L1PricingState().PosterDataCost(msg, msg.From(), *poster)
		posterCostInL2Gas := arbmath.BigToUintSaturating(arbmath.BigDiv(posterCost, header.BaseFee))
		*gascap = arbmath.SaturatingUAdd(*gascap, posterCostInL2Gas)
	}
}
