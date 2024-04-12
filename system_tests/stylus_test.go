// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build stylustest && !race
// +build stylustest,!race

package arbtest

import (
	"testing"
)

func TestProgramArbitratorKeccak(t *testing.T) {
	keccakTest(t, false)
}

func TestProgramArbitratorErrors(t *testing.T) {
	errorTest(t, false)
}

func TestProgramArbitratorStorage(t *testing.T) {
	storageTest(t, false)
}

func TestProgramArbitratorTransientStorage(t *testing.T) {
	transientStorageTest(t, false)
}

func TestProgramArbitratorCalls(t *testing.T) {
	testCalls(t, false)
}

func TestProgramArbitratorReturnData(t *testing.T) {
	testReturnData(t, false)
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

func TestProgramArbitratorActivateTwice(t *testing.T) {
	t.Parallel()
	testActivateTwice(t, false)
}

func TestProgramArbitratorActivateFails(t *testing.T) {
	t.Parallel()
	testActivateFails(t, false)
}

func TestProgramArbitratorEarlyExit(t *testing.T) {
	testEarlyExit(t, false)
}
