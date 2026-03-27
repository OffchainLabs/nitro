// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbostypes

import (
	"bytes"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/util"
)

// ParentChainPricingEntry holds gas and blob pricing data from a single parent chain block.
type ParentChainPricingEntry struct {
	BlockNumber   uint64
	BlockTimestamp uint64
	BlockHash     common.Hash
	L1BaseFee     *big.Int
	BlobBaseFee   *big.Int
	BlobGasUsed   uint64
	ExcessBlobGas uint64
}

// SerializeParentChainPricingBatchPayload encodes a batch of pricing entries.
// Wire format: entry count (uint64) followed by each entry's fields.
func SerializeParentChainPricingBatchPayload(entries []ParentChainPricingEntry) ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := util.Uint64ToWriter(uint64(len(entries)), wr); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := util.Uint64ToWriter(entry.BlockNumber, wr); err != nil {
			return nil, err
		}
		if err := util.Uint64ToWriter(entry.BlockTimestamp, wr); err != nil {
			return nil, err
		}
		if err := util.HashToWriter(entry.BlockHash, wr); err != nil {
			return nil, err
		}
		l1BaseFee := common.BigToHash(entry.L1BaseFee)
		if err := util.HashToWriter(l1BaseFee, wr); err != nil {
			return nil, err
		}
		blobBaseFee := common.BigToHash(entry.BlobBaseFee)
		if err := util.HashToWriter(blobBaseFee, wr); err != nil {
			return nil, err
		}
		if err := util.Uint64ToWriter(entry.BlobGasUsed, wr); err != nil {
			return nil, err
		}
		if err := util.Uint64ToWriter(entry.ExcessBlobGas, wr); err != nil {
			return nil, err
		}
	}
	return wr.Bytes(), nil
}

// ParseParentChainPricingBatchPayload decodes a batch of pricing entries.
func ParseParentChainPricingBatchPayload(rd io.Reader) ([]ParentChainPricingEntry, error) {
	count, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	entries := make([]ParentChainPricingEntry, count)
	for i := range count {
		blockNumber, err := util.Uint64FromReader(rd)
		if err != nil {
			return nil, err
		}
		blockTimestamp, err := util.Uint64FromReader(rd)
		if err != nil {
			return nil, err
		}
		blockHash, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		l1BaseFeeHash, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		blobBaseFeeHash, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		blobGasUsed, err := util.Uint64FromReader(rd)
		if err != nil {
			return nil, err
		}
		excessBlobGas, err := util.Uint64FromReader(rd)
		if err != nil {
			return nil, err
		}
		entries[i] = ParentChainPricingEntry{
			BlockNumber:   blockNumber,
			BlockTimestamp: blockTimestamp,
			BlockHash:     blockHash,
			L1BaseFee:     l1BaseFeeHash.Big(),
			BlobBaseFee:   blobBaseFeeHash.Big(),
			BlobGasUsed:   blobGasUsed,
			ExcessBlobGas: excessBlobGas,
		}
	}
	return entries, nil
}
