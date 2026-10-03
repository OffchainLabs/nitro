// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pointercheck

import "fmt"

type A struct {
	x, y int
}

// pointerCmp compares pointers, sometimes inside
func pointerCmp() {
	a, b := &A{}, &A{}
	// Simple comparions.
	if a != b { // want `comparison of two pointers in expression`
		fmt.Println("Not Equal")
	}
	if a == b { // want `comparison of two pointers in expression`
		fmt.Println("Equals")
	}
	// Nested binary expressions.
	if (2 > 1) && (a != b) { // want `comparison of two pointers in expression`
		fmt.Println("Still not equal")
	}
	if (174%15 > 3) && (2 > 1 && (1+2 > 2 || a != b)) { // want `comparison of two pointers in expression`
		fmt.Println("Who knows at this point")
	}
	// Nested and inside unary operator.
	if 10 > 5 && !(2 > 1 || a == b) { // want `comparison of two pointers in expression`
		fmt.Println("Not equal")
	}
	c, d := 1, 2
	if &c != &d {
		fmt.Println("Not equal")
	}
}

func legitCmps() {
	a, b := &A{}, &A{}
	if a.x == b.x {
		fmt.Println("Allowed")
	}
}

type cache struct {
	dirty *A
}

// matches does pointer comparison.
func (c *cache) matches(a *A) bool {
	return c.dirty == a // want `comparison of two pointers in expression`
}
