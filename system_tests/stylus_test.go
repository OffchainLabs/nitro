// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build stylustest
// +build stylustest

package arbtest

import (
	"testing"
)

func TestProgramArbitratorErrors(t *testing.T) {
	errorTest(t, false)
}

func TestProgramArbitratorStorage(t *testing.T) {
	storageTest(t, false)
}

func TestProgramArbitratorCalls(t *testing.T) {
	testCalls(t, false)
}

func TestProgramArbitratorLogs(t *testing.T) {
	testLogs(t, false)
}

func TestProgramArbitratorCreate(t *testing.T) {
	testCreate(t, false)
}

func TestProgramArbitratorEvmData(t *testing.T) {
	testEvmData(t, false)
}

func TestProgramArbitratorMemory(t *testing.T) {
	testMemory(t, false)
}
