// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !go1.18

package genericconf

func GetVersion() (string, string) {
	return "development", "development"
}
