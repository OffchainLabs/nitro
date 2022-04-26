// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package nodeInterface

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

type NodeInterfaceDebug struct {
	Address       addr
	backend       core.NodeInterfaceBackendAPI
	context       context.Context
	sourceMessage types.Message
	returnMessage struct {
		message *types.Message
		changed *bool
	}
}

func (n NodeInterfaceDebug) RetryableTimeoutQueue(c ctx, evm mech) (uint64, []bytes32, []uint64, error) {
	tickets := make([]common.Hash, 0)
	timeouts := make([]uint64, 0)

	closure := func(index uint64, ticket common.Hash) error {

		// we don't care if the retryable has expired
		retryable, err := c.State.RetryableState().OpenRetryable(ticket, 0)
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

	err := c.State.RetryableState().TimeoutQueue.ForEach(closure)
	tickets32 := make([]bytes32, len(tickets))
	for i, ticket := range tickets32 {
		tickets32[i] = bytes32(ticket)
	}
	return uint64(len(tickets)), tickets32, timeouts, err
}

type RetryableInfo = node_interfacegen.NodeInterfaceDebugRetryableInfo

func (n NodeInterfaceDebug) SerializeRetryable(c ctx, evm mech, ticket bytes32) (RetryableInfo, error) {
	// we don't care if the retryable has expired
	retryable, err := c.State.RetryableState().OpenRetryable(ticket, 0)
	if err != nil {
		return RetryableInfo{}, err
	}
	if retryable == nil {
		return RetryableInfo{}, fmt.Errorf("no retryable with id %v exists", ticket)
	}

	timeout, _ := retryable.CalculateTimeout()
	from, _ := retryable.From()
	toPointer, _ := retryable.To()
	callvalue, _ := retryable.Callvalue()
	beneficiary, _ := retryable.Beneficiary()
	calldata, _ := retryable.Calldata()
	tries, err := retryable.NumTries()

	to := common.Address{}
	if toPointer != nil {
		to = *toPointer
	}

	return node_interfacegen.NodeInterfaceDebugRetryableInfo{
		Timeout:     timeout,
		From:        from,
		To:          to,
		Value:       callvalue,
		Beneficiary: beneficiary,
		Tries:       tries,
		Data:        calldata,
	}, err
}
