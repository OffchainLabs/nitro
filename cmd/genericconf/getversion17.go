// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !go1.18

package genericconf

func Version() (string, string) {
	return "development", "development"
}
