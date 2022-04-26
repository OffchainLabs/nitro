// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package nodeInterface

import (
	"errors"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type NodeInterface struct {
	Address       addr
	backend       core.NodeInterfaceBackendAPI
	sourceMessage types.Message
	returnMessage *types.Message
}

func (n NodeInterface) FindBatchContainingBlock(c ctx, evm mech, blockNum uint64) (uint64, error) {
	return 0, errors.New("FindBatchContainingBlock unimplemented")
}

func (n NodeInterface) GetL1Confirmations(c ctx, evm mech, blockHash bytes32) (uint64, error) {
	return 0, errors.New("GetL1Confirmations unimplemented")
}

func (n NodeInterface) EstimateRetryableTicket(
	c ctx,
	evm mech,
	sender addr,
	deposit huge,
	to addr,
	l2CallValue huge,
	excessFeeRefundAddress addr,
	callValueRefundAddress addr,
	data []byte,
) error {
	return errors.New("EstimateRetryableTicket unimplemented")
}

func (n NodeInterface) ConstructOutboxProof(c ctx, evm mech, size, leaf uint64) (bytes32, bytes32, []bytes32, error) {
	return bytes32{}, bytes32{}, []bytes32{}, errors.New("ConstructOutboxProof unimplemented")
}
