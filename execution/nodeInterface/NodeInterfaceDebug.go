// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package nodeInterface

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/node_interfacegen"
)

type NodeInterfaceDebug struct {
	Address       addr
	backend       core.NodeInterfaceBackendAPI
	context       context.Context
	header        *types.Header
	sourceMessage *core.Message
	returnMessage struct {
		message *core.Message
		changed *bool
	}
}

type RetryableInfo = node_interfacegen.NodeInterfaceDebugRetryableInfo

func (n NodeInterfaceDebug) GetRetryable(c ctx, evm mech, ticket bytes32) (RetryableInfo, error) {
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
