// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package deferinloop

import "os"

// Bad: defer directly in for loop.
func forLoop() {
	for i := 0; i < 10; i++ {
		f, _ := os.Open("file")
		defer f.Close() // want `defer called directly in loop body, consider wrapping in an immediately-invoked function literal`
	}
}

// Bad: defer directly in range loop.
func rangeLoop() {
	for _, name := range []string{"a", "b"} {
		defer func() { _ = name }() // want `defer called directly in loop body, consider wrapping in an immediately-invoked function literal`
	}
}

// Good: defer inside anonymous function within loop.
func funcLitInLoop() {
	for i := 0; i < 10; i++ {
		func() {
			f, _ := os.Open("file")
			defer f.Close()
		}()
	}
}

// Good: defer outside of any loop.
func noLoop() {
	f, _ := os.Open("file")
	defer f.Close()
}

// Bad: defer in nested for loop.
func nestedLoops() {
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			defer func() { _, _ = i, j }() // want `defer called directly in loop body, consider wrapping in an immediately-invoked function literal`
		}
	}
}

// Good: defer inside func lit in nested loop.
func nestedLoopWithFunc() {
	for i := 0; i < 5; i++ {
		func() {
			for j := 0; j < 5; j++ {
				func() {
					defer func() { _, _ = i, j }()
				}()
			}
		}()
	}
}

// Bad: defer in range with nested func lit that also contains a loop.
func complexNesting() {
	for range []int{1, 2, 3} {
		defer os.Getenv("HOME") // want `defer called directly in loop body, consider wrapping in an immediately-invoked function literal`
		func() {
			for i := 0; i < 3; i++ {
				func() {
					defer func() { _ = i }()
				}()
			}
		}()
	}
}
