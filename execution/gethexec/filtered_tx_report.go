// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type FilterReasonType string

const (
	ReasonFrom                          FilterReasonType = "from"
	ReasonTo                            FilterReasonType = "to"
	ReasonDealiasedFrom                 FilterReasonType = "dealiased_from"
	ReasonRetryableBeneficiary          FilterReasonType = "retryable_beneficiary"
	ReasonRetryableFeeRefund            FilterReasonType = "retryable_fee_refund"
	ReasonRetryableTo                   FilterReasonType = "retryable_to"
	ReasonDealiasedRetryableBeneficiary FilterReasonType = "dealiased_retryable_beneficiary"
	ReasonDealiasedRetryableFeeRefund   FilterReasonType = "dealiased_retryable_fee_refund"
	ReasonEventRule                     FilterReasonType = "event_rule"
	ReasonContractAddress               FilterReasonType = "contract_address"
	ReasonContractCaller                FilterReasonType = "contract_caller"
	ReasonSelfdestructBeneficiary       FilterReasonType = "selfdestruct_beneficiary"
)

type RawLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

type FilterReason struct {
	Reason            FilterReasonType `json:"reason"`
	MatchedEvent      string           `json:"matchedEvent,omitempty"`
	MatchedTopicIndex int              `json:"matchedTopicIndex,omitempty"`
	RawLog            *RawLog          `json:"rawLog,omitempty"`
}

type FilteredAddressRecord struct {
	Address common.Address `json:"address"`
	FilterReason
}

type FilteredTxReport struct {
	Id                    string                  `json:"id"`
	TxHash                common.Hash             `json:"txHash"`
	TxRLP                 hexutil.Bytes           `json:"txRLP"`
	FilteredAddresses     []FilteredAddressRecord `json:"filteredAddresses"`
	BlockNumber           uint64                  `json:"blockNumber"`
	ParentBlockHash       common.Hash             `json:"parentBlockHash"`
	PositionInBlock       uint64                  `json:"positionInBlock"`
	FilteredAt            time.Time               `json:"filteredAt"`
	IsDelayed             bool                    `json:"isDelayed"`
	DelayedInboxRequestId string                  `json:"delayedInboxRequestId,omitempty"`
	ChainId               string                  `json:"chainId"`
}
