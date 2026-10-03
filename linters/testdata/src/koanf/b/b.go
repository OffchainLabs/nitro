// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package b

import (
	"flag"
	"fmt"
)

type ParCfg struct {
	child      ChildCfg      `koanf:"child"`
	grandChild GrandChildCfg `koanf:grandchild`
}

var defaultCfg = ParCfg{}

type ChildCfg struct {
	A bool `koanf:"A"`
	B bool `koanf:"B"`
	C bool `koanf:"C"`
	D bool `koanf:"D"` // want `field b.ChildCfg.D not used`
}

var defaultChildCfg = ChildCfg{}

func childConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".a", defaultChildCfg.A, "")
	f.Bool("b", defaultChildCfg.B, "")
	f.Bool("c", defaultChildCfg.C, "")
	f.Bool("d", defaultChildCfg.D, "")
}

type GrandChildCfg struct {
	A int `koanf:"A"` // want `field b.GrandChildCfg.A not used`
}

func (c *GrandChildCfg) Do() {
}

func configPtr() *ChildCfg {
	return nil
}
func config() ChildCfg {
	return ChildCfg{}
}

func init() {
	fmt.Printf("%v %v", config().A, configPtr().B)
	// This covers usage of both `ParCfg.Child` and `ChildCfg.C`.
	_ = defaultCfg.child.C
	// Covers usage of grandChild.
	defaultCfg.grandChild.Do()

}
