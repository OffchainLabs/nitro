//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package burn

import (
	"errors"
	"log"

	glog "github.com/ethereum/go-ethereum/log"
)

type Burner interface {
	Burn(amount uint64) error
	Restrict(err error)
}

type SystemBurner struct {
	gasBurnt uint64
}

func (burner *SystemBurner) Burn(amount uint64) error {
	burner.gasBurnt += amount
	return nil
}

func (burner *SystemBurner) Burned() uint64 {
	return burner.gasBurnt
}

func (burner *SystemBurner) Restrict(err error) {
	glog.Error("Restrict() received an error", "err", err)
}

type SafetyBurner struct {
	message string
	panics  bool
}

func NewSafetyBurner(message string, panics bool) *SafetyBurner {
	return &SafetyBurner{message, panics}
}

func (burner *SafetyBurner) Burn(amount uint64) error {
	if burner.panics {
		panic(burner.message)
	}
	return errors.New(burner.message)
}

func (burner *SafetyBurner) Restrict(err error) {
	log.Fatal("A metered burner was used for access-controlled work", err)
}
