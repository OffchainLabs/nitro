// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/arbmath"
)

var messageBuildFailedCounter = metrics.NewRegisteredCounter("arb/transactionfeed/message/buildfailed", nil)

type TransactionFeedMessageVersion uint32

const (
	TransactionFeedV1 TransactionFeedMessageVersion = 1
)

type TransactionFeedMessage struct {
	Version     TransactionFeedMessageVersion `json:"version"`
	TimestampMs uint64                        `json:"timestamp_ms"`
	PGARound    uint64                        `json:"pga_round"`
	Transaction IncludedTransaction           `json:"transaction"`
}

type IncludedTransaction struct {
	BlockNumber uint64            `json:"block_number"`
	TxIndex     uint32            `json:"tx_index"`
	RawTx       string            `json:"raw_tx"`
	TxHash      string            `json:"tx_hash"`
	Receipt     IncompleteReceipt `json:"receipt"`
}

type IncompleteReceipt struct {
	Status            uint8  `json:"status"`
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

func BuildFeedMessage(header *types.Header, tx *types.Transaction, receipt *types.Receipt) (*TransactionFeedMessage, error) {
	msg, err := buildFeedMessage(header, tx, receipt)
	if err != nil {
		messageBuildFailedCounter.Inc(1)
	}
	return msg, err
}

func buildFeedMessage(header *types.Header, tx *types.Transaction, receipt *types.Receipt) (*TransactionFeedMessage, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}
	if header == nil {
		return nil, fmt.Errorf("nil header for tx %s", tx.Hash().Hex())
	}
	if header.BaseFee == nil {
		return nil, fmt.Errorf("header missing BaseFee for tx %s at block %s", tx.Hash().Hex(), header.Number)
	}
	if receipt == nil {
		return nil, fmt.Errorf("nil receipt for tx %s", tx.Hash().Hex())
	}
	// This field should have been populated by FillTransaction
	if receipt.EffectiveGasPrice == nil {
		return nil, fmt.Errorf("receipt missing EffectiveGasPrice for tx %s", tx.Hash().Hex())
	}

	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("MarshalBinary for tx %s: %w", tx.Hash().Hex(), err)
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
		Version:     TransactionFeedV1,
		PGARound:    0, // TODO: placeholder until we connect with PGA round logic
		TimestampMs: arbmath.SaturatingUCast[uint64](time.Now().UnixMilli()),
		Transaction: IncludedTransaction{
			BlockNumber: header.Number.Uint64(),
			TxIndex:     arbmath.SaturatingUUCast[uint32](receipt.TransactionIndex),
			RawTx:       hexutil.Encode(rawTx),
			TxHash:      tx.Hash().Hex(),
			Receipt: IncompleteReceipt{
				Status:            arbmath.SaturatingUUCast[uint8](receipt.Status),
				GasUsed:           receipt.GasUsed,
				CumulativeGasUsed: receipt.CumulativeGasUsed,
				EffectiveGasPrice: hexutil.EncodeBig(receipt.EffectiveGasPrice),
				GasUsedForL1:      receipt.GasUsedForL1,
				BaseFee:           hexutil.EncodeBig(header.BaseFee),
				ContractAddress:   contractAddress,
				Logs:              logs,
			},
		},
	}, nil
}
