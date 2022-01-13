//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/burn"
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
	return initializeArbOS(stateDB, initData.AddressTableContents, initData.SendPartials, initData.DefaultAggregator, initData.RetryableData, initData.Accounts)
}

func initializeArbOS(
	stateDB *state.StateDB,
	addressTableContents []common.Address,
	sendPartials []common.Hash,
	defaultAggregator common.Address,
	retryableData []statetransfer.InitializationDataForRetryable,
	accounts []statetransfer.AccountInitializationInfo,
) error {
	arbosState, err := OpenArbosState(stateDB, &burn.SystemBurner{})
	if err != nil {
		return err
	}

	addrTable := arbosState.AddressTable()
	addrTableSize, err := addrTable.Size()
	if err != nil {
		return err
	}
	if addrTableSize != 0 {
		panic("address table must be empty")
	}
	for i, addr := range addressTableContents {
		slot, err := addrTable.Register(addr)
		if err != nil {
			return err
		}
		if uint64(i) != slot {
			panic("address table slot mismatch")
		}
	}

	err = merkleAccumulator.InitializeMerkleAccumulatorFromPartials(arbosState.backingStorage.OpenSubStorage(sendMerkleSubspace), sendPartials)
	if err != nil {
		return err
	}
	err = arbosState.L1PricingState().SetDefaultAggregator(defaultAggregator)
	if err != nil {
		return err
	}
	err = initializeRetryables(arbosState.RetryableState(), retryableData, 0)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		err = initializeAccount(stateDB, arbosState, account)
		if err != nil {
			return err
		}
	}

	return nil
}

func initializeRetryables(rs *retryables.RetryableState, data []statetransfer.InitializationDataForRetryable, currentTimestampToUse uint64) error {
	for _, r := range data {
		var to *common.Address
		if r.To != (common.Address{}) {
			to = &r.To
		}
		_, err := rs.CreateRetryable(currentTimestampToUse, r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
		if err != nil {
			return err
		}
	}
	return nil
}

func initializeAccount(statedb *state.StateDB, arbosState *ArbosState, account statetransfer.AccountInitializationInfo) error {
	l1pState := arbosState.L1PricingState()
	statedb.CreateAccount(account.Addr)
	statedb.SetNonce(account.Addr, account.Nonce)
	statedb.SetBalance(account.Addr, account.EthBalance)
	if account.ContractInfo != nil {
		statedb.SetCode(account.Addr, account.ContractInfo.Code)
		statedb.SetStorage(account.Addr, account.ContractInfo.ContractStorage)
	}
	if account.AggregatorInfo != nil {
		err := l1pState.SetAggregatorFeeCollector(account.Addr, account.AggregatorInfo.FeeCollector)
		if err != nil {
			return err
		}
		err = l1pState.SetFixedChargeForAggregatorL1Gas(account.Addr, account.AggregatorInfo.BaseFeeL1Gas)
		if err != nil {
			return err
		}
	}
	if account.AggregatorToPay != nil {
		err := l1pState.SetPreferredAggregator(account.Addr, *account.AggregatorToPay)
		if err != nil {
			return err
		}
	}

	return nil
}
