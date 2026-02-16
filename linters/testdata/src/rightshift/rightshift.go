// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package rightshift

import "fmt"

func doThing(v int) int {
	return 1 >> v // want `found rightshift \('1 >> x'\) expression, did you mean '1 << x' ?`
}

func calc() {
	val := 10
	fmt.Printf("%v", 1>>val) // want `found rightshift \('1 >> x'\) expression, did you mean '1 << x' ?`
	_ = doThing(1 >> val)    // want `found rightshift \('1 >> x'\) expression, did you mean '1 << x' ?`
	fmt.Printf("%v", 1<<val)
}
