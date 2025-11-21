// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package burn

import (
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/util"
)

type Burner interface {
	Burn(kind multigas.ResourceKind, amount uint64) error
	Burned() uint64
	GasLeft() uint64 // `SystemBurner`s panic (no notion of GasLeft)
	BurnOut() error
	Restrict(err error)
	HandleError(err error) error
	ReadOnly() bool
	TracingInfo() *util.TracingInfo
}

type SystemBurner struct {
	gasBurnt    multigas.MultiGas
	tracingInfo *util.TracingInfo
	readOnly    bool
}

func NewSystemBurner(tracingInfo *util.TracingInfo, readOnly bool) *SystemBurner {
	return &SystemBurner{
		tracingInfo: tracingInfo,
		readOnly:    readOnly,
	}
}

func (burner *SystemBurner) Burn(kind multigas.ResourceKind, amount uint64) error {
	burner.gasBurnt.SaturatingIncrementInto(kind, amount)
	return nil
}

func (burner *SystemBurner) Burned() uint64 {
	return burner.gasBurnt.SingleGas()
}

func (burner *SystemBurner) BurnOut() error {
	panic("called BurnOut on a system burner")
}

func (burner *SystemBurner) GasLeft() uint64 {
	panic("called GasLeft on a system burner")
}

func (burner *SystemBurner) Restrict(err error) {
	if err != nil {
		log.Error("Restrict() received an error", "err", err)
	}
}

func (burner *SystemBurner) HandleError(err error) error {
	panic(fmt.Sprintf("fatal error in system burner: %v", err))
}

func (burner *SystemBurner) ReadOnly() bool {
	return burner.readOnly
}

func (burner *SystemBurner) TracingInfo() *util.TracingInfo {
	return burner.tracingInfo
}
