// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionFeedMessageVersion uint32

const (
	TransactionFeedV1 TransactionFeedMessageVersion = 1
)

type TransactionFeedMessage struct {
	Version     uint32              `json:"version"`
	TimestampMs uint64              `json:"timestamp_ms"`
	PGARound    uint64              `json:"pga_round"`
	Transaction TransactionIncluded `json:"transaction"`
}

type TransactionIncluded struct {
	BlockNumber uint64            `json:"block_number"`
	TxIndex     uint32            `json:"tx_index"`
	RawTx       string            `json:"raw_tx"`
	TxHash      string            `json:"tx_hash"`
	Receipt     IncompleteReceipt `json:"receipt"`
}

type IncompleteReceipt struct {
	Status            uint64 `json:"status"`
	GasUsed           uint64 `json:"gas_used"`
	GasUsedForL1      uint64 `json:"gas_used_for_l1"`
	CumulativeGasUsed uint64 `json:"cumulative_gas_used"`
	EffectiveGasPrice string `json:"effective_gas_price"`
	BaseFee           string `json:"base_fee"`
	ContractAddress   string `json:"contract_address,omitempty"`
	Logs              []Log  `json:"logs"`
}

type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

func BuildFeedMessage(header *types.Header, tx *types.Transaction, receipt *types.Receipt, txIndex int) *TransactionFeedMessage {
	switch tx.Type() {
	case types.ArbitrumInternalTxType, types.ArbitrumRetryTxType:
		return nil
	}
	if header == nil || header.BaseFee == nil || receipt == nil {
		return nil
	}

	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return nil
	}

	effectiveGasPrice := receipt.EffectiveGasPrice
	if effectiveGasPrice == nil {
		effectiveGasPrice = header.BaseFee
	}

	var contractAddress string
	if receipt.ContractAddress != (common.Address{}) {
		contractAddress = receipt.ContractAddress.Hex()
	}

	logs := make([]Log, len(receipt.Logs))
	for i, l := range receipt.Logs {
		topics := make([]string, len(l.Topics))
		for j, t := range l.Topics {
			topics[j] = t.Hex()
		}
		logs[i] = Log{
			Address: l.Address.Hex(),
			Topics:  topics,
			Data:    hexutil.Encode(l.Data),
		}
	}

	return &TransactionFeedMessage{
		Version:     uint32(TransactionFeedV1),
		PGARound:    1, // TODO: placeholder until we connect with PGA round logic
		TimestampMs: uint64(time.Now().UnixMilli()),
		Transaction: TransactionIncluded{
			BlockNumber: header.Number.Uint64(),
			TxIndex:     uint32(txIndex),
			RawTx:       hexutil.Encode(rawTx),
			TxHash:      tx.Hash().Hex(),
			Receipt: IncompleteReceipt{
				Status:            receipt.Status,
				GasUsed:           receipt.GasUsed,
				CumulativeGasUsed: receipt.CumulativeGasUsed,
				EffectiveGasPrice: hexutil.EncodeBig(effectiveGasPrice),
				GasUsedForL1:      receipt.GasUsedForL1,
				BaseFee:           hexutil.EncodeBig(header.BaseFee),
				ContractAddress:   contractAddress,
				Logs:              logs,
			},
		},
	}
}
