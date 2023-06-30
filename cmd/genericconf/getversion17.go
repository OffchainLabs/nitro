// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

//go:build !go1.18

package genericconf

func GetVersion() (string, string) {
	return "development", "development"
}
