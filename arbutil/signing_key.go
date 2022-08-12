// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var keyIsHexRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]{64}$")

func LoadSigningKey(keyConfig string) (*[32]byte, error) {
	if keyConfig == "" {
		return nil, nil
	}
	keyIsHex := keyIsHexRegex.Match([]byte(keyConfig))
	var keyString string
	if keyIsHex {
		keyString = keyConfig
	} else {
		contents, err := os.ReadFile(keyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read signing key file: %w", err)
		}
		s := strings.TrimSpace(string(contents))
		if keyIsHexRegex.Match([]byte(s)) {
			keyString = s
		} else {
			return nil, errors.New("signing key file contents are not 32 bytes of hex")
		}
	}
	var b [32]byte = common.HexToHash(keyString)
	return &b, nil
}
