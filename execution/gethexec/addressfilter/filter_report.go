// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/arbitrum/filter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// lint:require-exhaustive-initialization
type DelayedReportData struct {
	InboxRequestId common.Hash `json:"delayedInboxRequestId"`
}

// lint:require-exhaustive-initialization
type FilterSetIdReport struct {
	FilterSetId uuid.UUID    `json:"filterSetId"`
	ChainId     *hexutil.Big `json:"chainId"`
	ReportedAt  time.Time    `json:"reportedAt"`
}

// lint:require-exhaustive-initialization
type FilteredTxReport struct {
	ID                string                         `json:"id"`
	TxHash            common.Hash                    `json:"txHash"`
	TxRLP             hexutil.Bytes                  `json:"txRLP"`
	FilteredAddresses []filter.FilteredAddressRecord `json:"filteredAddresses"`
	BlockNumber       uint64                         `json:"blockNumber"`
	ParentBlockHash   common.Hash                    `json:"parentBlockHash"`
	PositionInBlock   uint64                         `json:"positionInBlock"`
	FilteredAt        time.Time                      `json:"filteredAt"`
	IsDelayed         bool                           `json:"isDelayed"`
	*DelayedReportData
}
