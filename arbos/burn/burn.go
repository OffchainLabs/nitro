// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package burn

import (
	glog "github.com/ethereum/go-ethereum/log"
)

type Burner interface {
	Burn(amount uint64) error
	Burned() uint64
	Restrict(err error)
	ReadOnly() bool
}

type SystemBurner struct {
	gasBurnt uint64
	readOnly bool
}

func NewSystemBurner(readOnly bool) *SystemBurner {
	return &SystemBurner{
		readOnly: readOnly,
	}
}

func (burner *SystemBurner) Burn(amount uint64) error {
	burner.gasBurnt += amount
	return nil
}

func (burner *SystemBurner) Burned() uint64 {
	return burner.gasBurnt
}

func (burner *SystemBurner) Restrict(err error) {
	if err != nil {
		glog.Error("Restrict() received an error", "err", err)
	}
}

func (burner *SystemBurner) ReadOnly() bool {
	return burner.readOnly
}
