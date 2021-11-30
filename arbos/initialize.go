//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/retryables"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func InitializeArbosFromJSON(stateDB *state.StateDB, encoded []byte) error {
	initData := statetransfer.ArbosInitializationInfo{}
	err := json.Unmarshal(encoded, &initData)
	if err != nil {
		return err
	}
	initializeArbOS(stateDB, initData.AddressTableContents, initData.SendPartials, initData.DefaultAggregator, initData.RetryableData, initData.Accounts)
	return nil
}

func initializeArbOS(
	stateDB *state.StateDB,
	addressTableContents []common.Address,
	sendPartials []common.Hash,
	defaultAggregator common.Address,
	retryableData []statetransfer.InitializationDataForRetryable,
	accounts []statetransfer.AccountInitializationInfo,
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
	arbosState.L1PricingState().SetDefaultAggregator(defaultAggregator)
	initializeRetryables(arbosState.RetryableState(), retryableData, 0)
	for _, account := range accounts {
		initializeAccount(stateDB, arbosState, account)
	}
}

func initializeRetryables(rs *retryables.RetryableState, data []statetransfer.InitializationDataForRetryable, currentTimestampToUse uint64) {
	for _, r := range data {
		rs.CreateRetryable(0, r.Id, r.Timeout, r.From, r.To, r.Callvalue, r.Calldata)
	}
}

func initializeAccount(statedb *state.StateDB, arbosState *ArbosState, account statetransfer.AccountInitializationInfo) {
	l1pState := arbosState.L1PricingState()
	statedb.CreateAccount(account.Addr)
	statedb.SetNonce(account.Addr, account.Nonce)
	statedb.SetBalance(account.Addr, account.EthBalance)
	if account.ContractInfo != nil {
		statedb.SetCode(account.Addr, account.ContractInfo.Code)
		statedb.SetStorage(account.Addr, account.ContractInfo.ContractStorage)
	}
	if account.AggregatorInfo != nil {
		l1pState.SetAggregatorFeeCollector(account.Addr, account.AggregatorInfo.FeeCollector)
		l1pState.SetFixedChargeForAggregatorL1Gas(account.Addr, account.AggregatorInfo.BaseFeeL1Gas)
	}
	if account.AggregatorToPay != nil {
		l1pState.SetPreferredAggregator(account.Addr, *account.AggregatorToPay)
	}
}