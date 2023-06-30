// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package pretty

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func FirstFewBytes(b []byte) string {
	if len(b) < 9 {
		return fmt.Sprintf("[% x]", b)
	} else {
		return fmt.Sprintf("[% x ... ]", b[:8])
	}
}

func PrettyBytes(b []byte) string {
	hex := common.Bytes2Hex(b)
	if len(hex) > 24 {
		return fmt.Sprintf("%v...", hex[:24])
	}
	return hex
}

func PrettyHash(hash common.Hash) string {
	return FirstFewBytes(hash.Bytes())
}

func FirstFewChars(s string) string {
	if len(s) < 9 {
		return fmt.Sprintf("\"%s\"", s)
	} else {
		return fmt.Sprintf("\"%s...\"", s[:8])
	}
}
