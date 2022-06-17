// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package pretty

import "fmt"

func FirstFewBytes(b []byte) string {
	if len(b) < 9 {
		return fmt.Sprintf("[% x]", b)
	} else {
		return fmt.Sprintf("[% x ... ]", b[:8])
	}
}

func FirstFewChars(s string) string {
	if len(s) < 9 {
		return fmt.Sprintf("\"%s\"", s)
	} else {
		return fmt.Sprintf("\"%s...\"", s[:8])
	}
}
