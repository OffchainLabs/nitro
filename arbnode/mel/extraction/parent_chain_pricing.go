// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package melextraction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

const ParentChainPricingEpochSize = 32

// Blob gas pricing constants from EIP-4844 (Cancun).
var (
	minBlobBaseFee             = big.NewInt(1)
	blobBaseFeeUpdateFraction  = big.NewInt(3338477)
)

// collectPricingEntry creates a pricing entry from the parent chain block header.
func collectPricingEntry(header *types.Header) arbostypes.ParentChainPricingEntry {
	entry := arbostypes.ParentChainPricingEntry{
		BlockNumber:   header.Number.Uint64(),
		BlockTimestamp: header.Time,
		BlockHash:     header.Hash(),
		L1BaseFee:     new(big.Int).Set(header.BaseFee),
		BlobBaseFee:   big.NewInt(0),
		BlobGasUsed:   0,
		ExcessBlobGas: 0,
	}
	if header.BlobGasUsed != nil {
		entry.BlobGasUsed = *header.BlobGasUsed
	}
	if header.ExcessBlobGas != nil {
		entry.ExcessBlobGas = *header.ExcessBlobGas
		entry.BlobBaseFee = calcBlobBaseFee(*header.ExcessBlobGas)
	}
	return entry
}

// shouldFlushPricingEntries returns true if we have a full epoch of entries.
func shouldFlushPricingEntries(pending []arbostypes.ParentChainPricingEntry) bool {
	return len(pending) >= ParentChainPricingEpochSize
}

// createEpochPricingMessage creates a delayed message from batched pricing entries.
func createEpochPricingMessage(
	entries []arbostypes.ParentChainPricingEntry,
	parentChainHeader *types.Header,
	delayedMessagesSeen uint64,
) (*mel.DelayedInboxMessage, error) {
	payload, err := arbostypes.SerializeParentChainPricingBatchPayload(entries)
	if err != nil {
		return nil, err
	}
	lastEntry := entries[len(entries)-1]
	firstBlockNum := entries[0].BlockNumber
	requestId := crypto.Keccak256Hash([]byte("ParentChainPricingEpoch"), new(big.Int).SetUint64(firstBlockNum).Bytes())
	msg := &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_ParentChainPricingReport,
			Poster:      common.Address{},
			BlockNumber: lastEntry.BlockNumber,
			Timestamp:   lastEntry.BlockTimestamp,
			RequestId:   &requestId,
			L1BaseFee:   lastEntry.L1BaseFee,
		},
		L2msg: payload,
	}
	return &mel.DelayedInboxMessage{
		BlockHash:              parentChainHeader.Hash(),
		Message:                msg,
		ParentChainBlockNumber: parentChainHeader.Number.Uint64(),
	}, nil
}

// calcBlobBaseFee computes the EIP-4844 blob base fee from excess blob gas.
// Uses the Cancun formula: min_blob_base_fee * e^(excess_blob_gas / update_fraction)
// via the fake exponential approximation from EIP-4844.
func calcBlobBaseFee(excessBlobGas uint64) *big.Int {
	return fakeExponential(minBlobBaseFee, new(big.Int).SetUint64(excessBlobGas), blobBaseFeeUpdateFraction)
}

// fakeExponential approximates factor * e ** (numerator / denominator) using
// Taylor expansion as described in EIP-4844.
func fakeExponential(factor, numerator, denominator *big.Int) *big.Int {
	i := new(big.Int).SetUint64(1)
	output := new(big.Int)
	numeratorAccum := new(big.Int).Set(factor)
	numeratorAccum.Mul(numeratorAccum, denominator)
	for numeratorAccum.Sign() > 0 {
		output.Add(output, numeratorAccum)
		numeratorAccum.Mul(numeratorAccum, numerator)
		numeratorAccum.Div(numeratorAccum, denominator)
		numeratorAccum.Div(numeratorAccum, i)
		i.Add(i, big.NewInt(1))
	}
	output.Div(output, denominator)
	return output
}
