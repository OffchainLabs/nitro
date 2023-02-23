// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package burn

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	glog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/util"
)

const StorageReadCostV0 = params.SloadGasEIP2200
const StorageWriteCostV0 = params.SstoreSetGasEIP2200
const StorageWriteZeroCostV0 = params.SstoreResetGasEIP2200

type Burner interface {
	Burn(amount uint64) error
	Burned() uint64
	ChargeForRead(db vm.StateDB, key common.Hash) error
	ChargeForWrite(db vm.StateDB, key, value common.Hash) error
	Restrict(err error)
	HandleError(err error) error
	ReadOnly() bool
	IsSystem() bool
	TracingInfo() *util.TracingInfo
	SetVersion(version uint64)
	Version() uint64
}

type SystemBurner struct {
	gasBurnt    uint64
	tracingInfo *util.TracingInfo
	readOnly    bool
	version     uint64 // set during OpenArbosState
}

func NewSystemBurner(tracingInfo *util.TracingInfo, readOnly bool) *SystemBurner {
	return &SystemBurner{
		tracingInfo: tracingInfo,
		readOnly:    readOnly,
	}
}

func (burner *SystemBurner) Burn(amount uint64) error {
	burner.gasBurnt += amount
	return nil
}

func (burner *SystemBurner) Burned() uint64 {
	return burner.gasBurnt
}

func (burner *SystemBurner) ChargeForRead(db vm.StateDB, key common.Hash) error {
	burner.gasBurnt += StorageReadCostV0
	return nil
}

func (burner *SystemBurner) ChargeForWrite(db vm.StateDB, key, value common.Hash) error {
	if value == (common.Hash{}) {
		burner.gasBurnt += StorageWriteZeroCostV0
	} else {
		burner.gasBurnt += StorageWriteCostV0
	}
	return nil
}

func (burner *SystemBurner) Restrict(err error) {
	if err != nil {
		glog.Error("Restrict() received an error", "err", err)
	}
}

func (burner *SystemBurner) HandleError(err error) error {
	panic(fmt.Sprintf("fatal error in system burner: %v", err))
}

func (burner *SystemBurner) ReadOnly() bool {
	return burner.readOnly
}

func (burner *SystemBurner) IsSystem() bool {
	return true
}

func (burner *SystemBurner) TracingInfo() *util.TracingInfo {
	return burner.tracingInfo
}

func (burner *SystemBurner) SetVersion(version uint64) {
	burner.version = version
}

func (burner *SystemBurner) Version() uint64 {
	return burner.version
}
