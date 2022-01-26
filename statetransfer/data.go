//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ArbosInitializationInfo struct {
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
	BaseFeeL1Gas *big.Int
}
