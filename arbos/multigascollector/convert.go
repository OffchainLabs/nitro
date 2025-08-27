// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package multigascollector

import (
	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
)

// ToProto converts the TransactionMultiGas to its protobuf representation.
func (tx *TransactionMultiGas) ToProto() *proto.TransactionMultiGasData {
	multiGasData := &proto.MultiGasData{
		Computation:   tx.MultiGas.Get(multigas.ResourceKindComputation),
		StorageAccess: tx.MultiGas.Get(multigas.ResourceKindStorageAccess),
		StorageGrowth: tx.MultiGas.Get(multigas.ResourceKindStorageGrowth),
		HistoryGrowth: tx.MultiGas.Get(multigas.ResourceKindHistoryGrowth),
		L1Calldata:    tx.MultiGas.Get(multigas.ResourceKindL1Calldata),
		L2Calldata:    tx.MultiGas.Get(multigas.ResourceKindL2Calldata),
	}

	if unknown := tx.MultiGas.Get(multigas.ResourceKindUnknown); unknown > 0 {
		multiGasData.Unknown = &unknown
	}

	if refund := tx.MultiGas.GetRefund(); refund > 0 {
		multiGasData.Refund = &refund
	}

	return &proto.TransactionMultiGasData{
		TxHash:   tx.TxHash,
		TxIndex:  tx.TxIndex,
		MultiGas: multiGasData,
	}
}

// ToProto converts the BlockInfo to its protobuf representation.
func (btmg *BlockInfo) ToProto() *proto.BlockMultiGasData {
	return &proto.BlockMultiGasData{
		BlockNumber:    btmg.BlockNumber,
		BlockHash:      btmg.BlockHash,
		BlockTimestamp: btmg.BlockTimestamp,
	}
}
