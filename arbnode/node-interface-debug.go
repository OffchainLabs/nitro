// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/pkg/errors"
)

func ApplyNodeInterfaceDebug(
	msg Message,
	ctx context.Context,
	statedb *state.StateDB,
	backend core.NodeInterfaceBackendAPI,
	nodeInterfaceDebug abi.ABI,
) (Message, *ExecutionResult, error) {

	queueMethod := nodeInterfaceDebug.Methods["retryableTimeoutQueue"]
	retryMethod := nodeInterfaceDebug.Methods["serializeRetryable"]

	calldata := msg.Data()
	if len(calldata) < 4 {
		return msg, nil, errors.New("calldata for NodeInterfaceDebug.sol is too short")
	}

	state, err := arbosState.OpenSystemArbosState(statedb, nil, true)
	if err != nil {
		log.Error("failed to open ArbOS state", "err", err)
		return msg, nil, fmt.Errorf("failed to open ArbOS state %w", err)
	}

	if bytes.Equal(queueMethod.ID, calldata[:4]) {
		_, err := queueMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		res, err := nodeInterfaceDebugTimeoutQueue(state, queueMethod)
		return msg, res, err
	}

	if bytes.Equal(retryMethod.ID, calldata[:4]) {
		inputs, err := retryMethod.Inputs.Unpack(calldata[4:])
		if err != nil {
			return msg, nil, err
		}
		id, _ := inputs[0].([32]byte)
		res, err := nodeInterfaceDebugSerializeRetryable(state, id)
		return msg, res, err
	}

	return msg, nil, nil
}

func nodeInterfaceDebugTimeoutQueue(state *arbosState.ArbosState, method abi.Method) (*ExecutionResult, error) {

	tickets := make([]common.Hash, 0)
	timeouts := make([]uint64, 0)

	closure := func(index uint64, ticket common.Hash) error {

		// we don't care if the retryable has expired
		retryable, err := state.RetryableState().OpenRetryable(ticket, 0)
		if err != nil {
			return err
		}
		timeout, err := retryable.CalculateTimeout()
		if err != nil {
			return err
		}
		windows, err := retryable.TimeoutWindowsLeft()
		if err != nil {
			return err
		}
		timeout -= windows * retryables.RetryableLifetimeSeconds

		tickets = append(tickets, ticket)
		timeouts = append(timeouts, timeout)
		return nil
	}
	err := state.RetryableState().TimeoutQueue.ForEach(closure)
	if err != nil {
		return nil, err
	}

	returnData, err := method.Outputs.Pack(len(tickets), tickets, timeouts)
	if err != nil {
		return nil, fmt.Errorf("internal error: failed to encode outputs: %w", err)
	}
	res := &ExecutionResult{
		UsedGas:       0,
		Err:           nil,
		ReturnData:    returnData,
		ScheduledTxes: nil,
	}
	return res, nil
}

func nodeInterfaceDebugSerializeRetryable(state *arbosState.ArbosState, id common.Hash) (*ExecutionResult, error) {
	// we don't care if the retryable has expired
	retryable, err := state.RetryableState().OpenRetryable(id, 0)
	if err != nil {
		return nil, err
	}
	if retryable == nil {
		return nil, fmt.Errorf("no retryable with id %v exists", id)
	}
	returnData, err := retryable.SerializeRetryable()
	res := &ExecutionResult{
		UsedGas:       0,
		Err:           nil,
		ReturnData:    returnData,
		ScheduledTxes: nil,
	}
	return res, err
}
