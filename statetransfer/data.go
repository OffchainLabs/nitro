//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type ArbosInitializationInfo struct {
	AddressTableContents []common.Address
	SendPartials         []common.Hash
	DefaultAggregator    common.Address
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
}

type AccountInitContractInfo struct {
	Code            []byte
	ContractStorage map[common.Hash]common.Hash
}

type AccountInitAggregatorInfo struct {
	FeeCollector common.Address
	BaseFeeL1Gas *big.Int
}

