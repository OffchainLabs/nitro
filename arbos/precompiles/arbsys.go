//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

type ArbSys struct {}

func (con ArbSys) ArbBlockNumber(st *state.StateDB) {}
func (con ArbSys) ArbChainID(st *state.StateDB) {}
func (con ArbSys) ArbOSVersion() {}
func (con ArbSys) GetStorageAt(st *state.StateDB, address common.Address, index *big.Int) {}
func (con ArbSys) GetStorageGasAvailable(st *state.StateDB) {}
func (con ArbSys) GetTransactionCount(st *state.StateDB, account common.Address) {}
func (con ArbSys) IsTopLevelCall(st *state.StateDB) {}
func (con ArbSys) MapL1SenderContractAddressToL2Alias(sender common.Address, dest common.Address) {}
func (con ArbSys) MyCallersAddressWithoutAliasing(st *state.StateDB) {}
func (con ArbSys) SendTxToL1(st *state.StateDB, value *big.Int, destination common.Address, calldataForL1 []byte) {}
func (con ArbSys) WasMyCallersAddressAliased(st *state.StateDB) {}
func (con ArbSys) WithdrawEth(st *state.StateDB, value *big.Int, destination common.Address) {}
