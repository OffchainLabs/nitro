// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var ownershipTests = []systest.Scenario{
	systest.Test(testRunChainOwnerManagement,
		systest.MatrixArbOS(params.ArbosVersion_51, params.ArbosVersion_60),
		systest.WithArbOSInit(params.ArbOSInit{NativeTokenSupplyManagementEnabled: true})),
	systest.Test(testRunNativeTokenOwnerManagement,
		systest.MatrixArbOS(params.ArbosVersion_51, params.ArbosVersion_60),
		systest.WithArbOSInit(params.ArbOSInit{NativeTokenSupplyManagementEnabled: true})),
}

func testRunChainOwnerManagement(env *systest.Env) {
	runOwnerManagement(env, "ChainOwner")
}

func testRunNativeTokenOwnerManagement(env *systest.Env) {
	runOwnerManagement(env, "NativeTokenOwner")
}

func runOwnerManagement(env *systest.Env, ownerType string) {
	auth := env.L2.TransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	// Get event topics
	arbOwnerABI, err := precompilesgen.ArbOwnerMetaData.GetAbi()
	env.Require(err)

	var addedTopic, removedTopic common.Hash
	var parseAdded, parseRemoved func(types.Log) (any, error)

	if ownerType == "ChainOwner" {
		addedTopic = arbOwnerABI.Events["ChainOwnerAdded"].ID
		removedTopic = arbOwnerABI.Events["ChainOwnerRemoved"].ID
		parseAdded = func(log types.Log) (any, error) { return arbOwner.ParseChainOwnerAdded(log) }
		parseRemoved = func(log types.Log) (any, error) { return arbOwner.ParseChainOwnerRemoved(log) }
	} else {
		addedTopic = arbOwnerABI.Events["NativeTokenOwnerAdded"].ID
		removedTopic = arbOwnerABI.Events["NativeTokenOwnerRemoved"].ID
		parseAdded = func(log types.Log) (any, error) { return arbOwner.ParseNativeTokenOwnerAdded(log) }
		parseRemoved = func(log types.Log) (any, error) { return arbOwner.ParseNativeTokenOwnerRemoved(log) }
	}

	// Create test account
	env.L2.Info.GenerateAccount("TestOwner")
	testAddr := env.L2.Info.GetAddress("TestOwner")

	// 1. Check if account is NOT an owner
	isOwner, err := checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	env.Require(err)
	env.True(!isOwner, "account should not be owner initially")

	// 2. Add account as owner
	tx, err := addOwner(arbOwner, &auth, ownerType, testAddr)
	env.Require(err)
	receipt := env.L2.EnsureTxSucceeded(tx)

	// 3. Verify Added event was emitted (only for ArbOS 60)
	shouldEmit := env.Spec.ArbOSVersion.UnwrapOr(0) >= params.ArbosVersion_60
	foundEvent := findAndParseEvent(receipt.Logs, addedTopic, parseAdded, testAddr)
	env.Equal(shouldEmit, foundEvent, "%sAdded event emission mismatch", ownerType)

	// 4. Check if account IS an owner
	isOwner, err = checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	env.Require(err)
	env.True(isOwner, "account should be owner after adding")

	// 5. Remove account as owner
	tx, err = removeOwner(arbOwner, &auth, ownerType, testAddr)
	env.Require(err)
	receipt = env.L2.EnsureTxSucceeded(tx)

	// 6. Verify Removed event was emitted (only for ArbOS 60)
	foundEvent = findAndParseEvent(receipt.Logs, removedTopic, parseRemoved, testAddr)
	env.Equal(shouldEmit, foundEvent, "%sRemoved event emission mismatch", ownerType)

	// 7. Check if account is NOT an owner
	isOwner, err = checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	env.Require(err)
	env.True(!isOwner, "account should not be owner after removal")
}

func checkIsOwner(arbOwner *precompilesgen.ArbOwner, callOpts *bind.CallOpts, ownerType string, addr common.Address) (bool, error) {
	if ownerType == "ChainOwner" {
		return arbOwner.IsChainOwner(callOpts, addr)
	}
	return arbOwner.IsNativeTokenOwner(callOpts, addr)
}

func addOwner(arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts, ownerType string, addr common.Address) (*types.Transaction, error) {
	if ownerType == "ChainOwner" {
		return arbOwner.AddChainOwner(auth, addr)
	}
	return arbOwner.AddNativeTokenOwner(auth, addr)
}

func removeOwner(arbOwner *precompilesgen.ArbOwner, auth *bind.TransactOpts, ownerType string, addr common.Address) (*types.Transaction, error) {
	if ownerType == "ChainOwner" {
		return arbOwner.RemoveChainOwner(auth, addr)
	}
	return arbOwner.RemoveNativeTokenOwner(auth, addr)
}

func findAndParseEvent(logs []*types.Log, topic common.Hash, parseFunc func(types.Log) (any, error), expectedAddr common.Address) bool {
	for _, lg := range logs {
		if lg.Topics[0] != topic {
			continue
		}
		ev, err := parseFunc(*lg)
		if err != nil {
			continue
		}
		switch event := ev.(type) {
		case *precompilesgen.ArbOwnerChainOwnerAdded:
			if event.Owner == expectedAddr {
				return true
			}
		case *precompilesgen.ArbOwnerChainOwnerRemoved:
			if event.Owner == expectedAddr {
				return true
			}
		case *precompilesgen.ArbOwnerNativeTokenOwnerAdded:
			if event.Owner == expectedAddr {
				return true
			}
		case *precompilesgen.ArbOwnerNativeTokenOwnerRemoved:
			if event.Owner == expectedAddr {
				return true
			}
		}
	}
	return false
}
