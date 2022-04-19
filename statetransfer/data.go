// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ArbosInitializationInfo struct {
	Blocks               []StoredBlock
	AddressTableContents []common.Address
	RetryableData        []InitializationDataForRetryable
	Accounts             []AccountInitializationInfo
}

type InitializationDataForRetryable struct {
	Id          common.Hash
	Timeout     uint64
	From        common.Address
	To          common.Address
	Callvalue   *big.Int
	Beneficiary common.Address
	Calldata    []byte
}

type AccountInitializationInfo struct {
	Addr            common.Address
	Nonce           uint64
	EthBalance      *big.Int
	ContractInfo    *AccountInitContractInfo
	AggregatorInfo  *AccountInitAggregatorInfo
	AggregatorToPay *common.Address
	ClassicHash     common.Hash
}

type AccountInitContractInfo struct {
	Code            []byte
	ContractStorage map[common.Hash]common.Hash
}

type AccountInitAggregatorInfo struct {
	FeeCollector common.Address
	BaseFeeL1Gas *big.Int // This is unused in Nitro, so its value will be ignored.
}

type StoredBlock struct {
	Header       types.Header
	Transactions []types.ArbitrumLegacyTransactionResult `json:"transactions" gencodec:"required"`
	Reciepts     types.Receipts                          `json:"reciepts" gencodec:"required"`
}
