// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package burn

import (
	"fmt"

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
	RequireGas(amount uint64) error
	Restrict(err error)
	HandleError(err error) error
	ReadOnly() bool
	OutsideTx() bool
	TracingInfo() *util.TracingInfo
	SetVersion(version uint64)
	Version() uint64
}

type SystemBurner struct {
	gasBurnt    uint64
	tracingInfo *util.TracingInfo
	outsideTx   bool
	readOnly    bool
	version     uint64 // set during OpenArbosState
}

func NewSystemBurner(tracingInfo *util.TracingInfo, outsideTx bool) *SystemBurner {
	return &SystemBurner{
		tracingInfo: tracingInfo,
		outsideTx:   outsideTx,
		readOnly:    outsideTx,
	}
}

func NewSystemBurnerWrite(tracingInfo *util.TracingInfo) *SystemBurner {
	return &SystemBurner{
		tracingInfo: tracingInfo,
		outsideTx:   true,
		readOnly:    false,
	}
}

func (burner *SystemBurner) Burn(amount uint64) error {
	burner.gasBurnt += amount
	return nil
}

func (burner *SystemBurner) Burned() uint64 {
	return burner.gasBurnt
}

func (burner *SystemBurner) RequireGas(amount uint64) error {
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

func (burner *SystemBurner) OutsideTx() bool {
	return burner.outsideTx
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
