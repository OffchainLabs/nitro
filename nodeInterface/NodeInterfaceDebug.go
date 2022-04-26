// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package nodeInterface

import (
	"errors"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type NodeInterfaceDebug struct {
	Address       addr
	backend       core.NodeInterfaceBackendAPI
	sourceMessage types.Message
	returnMessage *types.Message
}

func (n NodeInterfaceDebug) RetryableTimeoutQueue(c ctx, evm mech) (uint64, []bytes32, []uint64, error) {
	return 0, []bytes32{}, []uint64{}, errors.New("RetryableTimeoutQueue unimplemented")
}

func (n NodeInterfaceDebug) SerializeRetryable(c ctx, evm mech, ticket bytes32) ([]byte, error) {
	return []byte{}, errors.New("SerializeRetryable unimplemented")
}
