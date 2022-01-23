//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package burn

import (
	glog "github.com/ethereum/go-ethereum/log"
)

type Burner interface {
	Burn(amount uint64) error
	Restrict(err error)
	ReadOnly() bool
}

type SystemBurner struct {
	gasBurnt uint64
	write    bool
}

func NewSystemBurner(write bool) *SystemBurner {
	return &SystemBurner{
		write: write,
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
	return !burner.write
}
