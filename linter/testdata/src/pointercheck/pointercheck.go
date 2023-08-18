package pointercheck

import "fmt"

type A struct {
	x, y int
}

// pointerCmp compares pointers, sometimes inside
func pointerCmp() {
	a, b := &A{}, &A{}
	// Simple comparions.
	if a != b {
		fmt.Println("Not Equal")
	}
	if a == b {
		fmt.Println("Equals")
	}
	// Nested binary expressions.
	if (2 > 1) && (a != b) {
		fmt.Println("Still not equal")
	}
	if (174%15 > 3) && (2 > 1 && (1+2 > 2 || a != b)) {
		fmt.Println("Who knows at this point")
	}
	// Nested and inside unary operator.
	if 10 > 5 && !(2 > 1 || a == b) {
		fmt.Println("Not equal")
	}
	c, d := 1, 2
	if &c != &d {
		fmt.Println("Not equal")
	}
}
