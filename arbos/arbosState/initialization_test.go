//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbosState

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/arbos/merkleAccumulator"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/statetransfer"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	prand := util.NewPseudoRandomDataSource(1)
	tryMarshalUnmarshal(
		&statetransfer.ArbosInitializationInfo{
			AddressTableContents: []common.Address{prand.GetAddress()},
			SendPartials:         []common.Hash{prand.GetHash(), prand.GetHash()},
			DefaultAggregator:    prand.GetAddress(),
			RetryableData:        []statetransfer.InitializationDataForRetryable{pseudorandomRetryableInitForTesting(prand)},
			Accounts:             []statetransfer.AccountInitializationInfo{pseudorandomAccountInitInfoForTesting(prand)},
		},
		t,
	)
}

func tryMarshalUnmarshal(input *statetransfer.ArbosInitializationInfo, t *testing.T) {
	marshaled, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(marshaled) {
		t.Fatal()
	}
	if len(marshaled) == 0 {
		t.Fatal()
	}

	output := statetransfer.ArbosInitializationInfo{}
	err = json.Unmarshal(marshaled, &output)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.AddressTableContents) != 1 {
		t.Fatal(output)
	}

	genesisAlloc, err := GetGenesisAllocFromJSON(marshaled)
	Require(t, err)
	genesis := core.Genesis{Alloc: genesisAlloc}

	raw := rawdb.NewMemoryDatabase()
	Require(t, err)

	block, err := genesis.Commit(raw)
	Require(t, err)
	stateDb, err := state.New(block.Header().Root, state.NewDatabase(raw), nil)
	Require(t, err)

	arbState, err := OpenArbosState(stateDb, &burn.SystemBurner{})
	Require(t, err)
	checkAddressTable(arbState, input.AddressTableContents, t)
	checkSendAccum(arbState, input.SendPartials, t)
	checkDefaultAgg(arbState, input.DefaultAggregator, t)
	checkRetryables(arbState, input.RetryableData, t)
	checkAccounts(stateDb, arbState, input.Accounts, t)
}

func pseudorandomRetryableInitForTesting(prand *util.PseudoRandomDataSource) statetransfer.InitializationDataForRetryable {
	return statetransfer.InitializationDataForRetryable{
		Id:          prand.GetHash(),
		Timeout:     prand.GetUint64(),
		From:        prand.GetAddress(),
		To:          prand.GetAddress(),
		Callvalue:   prand.GetHash().Big(),
		Beneficiary: prand.GetAddress(),
		Calldata:    prand.GetData(256),
	}
}

func pseudorandomAccountInitInfoForTesting(prand *util.PseudoRandomDataSource) statetransfer.AccountInitializationInfo {
	aggToPay := prand.GetAddress()
	return statetransfer.AccountInitializationInfo{
		Addr:       prand.GetAddress(),
		Nonce:      prand.GetUint64(),
		EthBalance: prand.GetHash().Big(),
		ContractInfo: &statetransfer.AccountInitContractInfo{
			Code:            prand.GetData(256),
			ContractStorage: pseudorandomHashHashMapForTesting(prand, 16),
		},
		AggregatorInfo: &statetransfer.AccountInitAggregatorInfo{
			FeeCollector: prand.GetAddress(),
			BaseFeeL1Gas: prand.GetHash().Big(),
		},
		AggregatorToPay: &aggToPay,
	}
}

func pseudorandomHashHashMapForTesting(prand *util.PseudoRandomDataSource, maxItems uint64) map[common.Hash]common.Hash {
	size := int(prand.GetUint64() % maxItems)
	ret := make(map[common.Hash]common.Hash)
	for i := 0; i < size; i++ {
		ret[prand.GetHash()] = prand.GetHash()
	}
	return ret
}

func checkAddressTable(arbState *ArbosState, addrTable []common.Address, t *testing.T) {
	atab := arbState.AddressTable()
	atabSize, err := atab.Size()
	Require(t, err)
	if atabSize != uint64(len(addrTable)) {
		Fail(t)
	}
	for i, addr := range addrTable {
		res, exists, err := atab.LookupIndex(uint64(i))
		Require(t, err)
		if !exists {
			Fail(t)
		}
		if res != addr {
			Fail(t)
		}
	}
}

func checkSendAccum(arbState *ArbosState, expected []common.Hash, t *testing.T) {
	sa := arbState.SendMerkleAccumulator()
	partials, err := sa.GetPartials()
	Require(t, err)
	if len(partials) != len(expected) {
		t.Fatal()
	}
	pexp := make([]*common.Hash, len(expected))
	for i, partial := range partials {
		if *partial != expected[i] {
			t.Fatal()
		}
		pexp[i] = &expected[i]
	}
	acc2, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(pexp)
	Require(t, err)
	a2Root, err := acc2.Root()
	Require(t, err)
	saRoot, err := sa.Root()
	Require(t, err)
	if a2Root != saRoot {
		t.Fatal()
	}
}

func checkDefaultAgg(arbState *ArbosState, expected common.Address, t *testing.T) {
	da, err := arbState.L1PricingState().DefaultAggregator()
	Require(t, err)
	if da != expected {
		Fail(t)
	}
}

func checkRetryables(arbState *ArbosState, expected []statetransfer.InitializationDataForRetryable, t *testing.T) {
	ret := arbState.RetryableState()
	for _, exp := range expected {
		found, err := ret.OpenRetryable(exp.Id, 0)
		Require(t, err)
		if found == nil {
			Fail(t)
		}
		// TODO: detailed comparison
	}
}

func checkAccounts(db *state.StateDB, arbState *ArbosState, accts []statetransfer.AccountInitializationInfo, t *testing.T) {
	l1p := arbState.L1PricingState()
	for _, acct := range accts {
		addr := acct.Addr
		if db.GetNonce(addr) != acct.Nonce {
			t.Fatal()
		}
		if db.GetBalance(addr).Cmp(acct.EthBalance) != 0 {
			t.Fatal()
		}
		if acct.ContractInfo != nil {
			if !bytes.Equal(acct.ContractInfo.Code, db.GetCode(addr)) {
				t.Fatal()
			}
			err := db.ForEachStorage(addr, func(key common.Hash, value common.Hash) bool {
				val2, exists := acct.ContractInfo.ContractStorage[key]
				if !exists {
					t.Fatal()
				}
				if value != val2 {
					t.Fatal()
				}
				return false
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if acct.AggregatorInfo != nil {
			fc, err := l1p.AggregatorFeeCollector(addr)
			Require(t, err)
			if fc != acct.AggregatorInfo.FeeCollector {
				t.Fatal()
			}
			charge, err := l1p.FixedChargeForAggregatorL1Gas(addr)
			Require(t, err)
			if charge.Cmp(acct.AggregatorInfo.BaseFeeL1Gas) != 0 {
				Fail(t)
			}
		}
		if acct.AggregatorToPay != nil {
			prefAgg, _, err := l1p.PreferredAggregator(addr)
			Require(t, err)
			if prefAgg != *acct.AggregatorToPay {
				Fail(t)
			}
		}
	}
	_ = l1p
}
