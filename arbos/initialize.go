//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/l1pricing"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"math/big"
)

func InitializeArbOS(
	stateDB *state.StateDB,
	addressTableContents []common.Address,
	sendPartials []common.Hash,
	l1Data *L1PricingInitializationData,
	retryableData []InitializationDataForRetryable,
) {
	arbosState := OpenArbosState(stateDB)

	addrTable := arbosState.AddressTable()
	if addrTable.Size() != 0 {
		panic("address table must be empty")
	}
	for i, addr := range addressTableContents {
		slot := addrTable.Register(addr)
		if uint64(i) != slot {
			panic("address table slot mismatch")
		}
	}

	merkleAccumulator.InitializeMerkleAccumulatorFromPartials(arbosState.backingStorage.OpenSubStorage(sendMerkleSubspace), sendPartials)

	initializeL1Pricing(arbosState.L1PricingState(), l1Data)

	initializeRetryables(arbosState.RetryableState(), retryableData, 0)
}

func InitializeArbosFromJSON(stateDB *state.StateDB, encoded []byte) error {
	initData := ArbosInitializationInfo{}
	err := json.Unmarshal(encoded, &initData)
	if err != nil {
		return err
	}
	InitializeArbOS(stateDB, initData.addressTableContents, initData.sendPartials, initData.l1Data, initData.retryableData)
	return nil
}

type ArbosInitializationInfo struct {
	addressTableContents []common.Address
	sendPartials []common.Hash
	l1Data *L1PricingInitializationData
	retryableData []InitializationDataForRetryable
}

type L1PricingInitializationData struct {
	defaultAggregator           common.Address
	preferredAggregators        map[common.Address]common.Address
	aggregatorFixedCharges      map[common.Address]*big.Int
	aggregatorFeeCollectors     map[common.Address]common.Address
	aggregatorCompressionRatios map[common.Address]uint64
}

func initializeL1Pricing(l1p *l1pricing.L1PricingState, data *L1PricingInitializationData) {
	l1p.SetDefaultAggregator(data.defaultAggregator)
	for a, b := range data.preferredAggregators {
		l1p.SetPreferredAggregator(a, b)
	}
	for a, b := range data.aggregatorFixedCharges {
		l1p.SetFixedChargeForAggregatorL1Gas(a, b)
	}
	for a, b := range data.aggregatorFeeCollectors {
		l1p.SetAggregatorFeeCollector(a, b)
	}
	for a, b := range data.aggregatorCompressionRatios {
		l1p.SetAggregatorCompressionRatio(a, &b)
	}
}

type InitializationDataForRetryable struct {
	id   common.Hash
	timeout uint64
	from common.Address
	to   common.Address
	callvalue *big.Int
	calldata []byte
}

func initializeRetryables(rs *retryables.RetryableState, data []InitializationDataForRetryable, currentTimestampToUse uint64) {
	for _, r := range data {
		rs.CreateRetryable(0, r.id, r.timeout, r.from, r.to, r.callvalue, r.calldata)
	}
}