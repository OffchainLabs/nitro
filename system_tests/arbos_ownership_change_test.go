// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/stretchr/testify/require"
)

func TestChainOwnerManagement(t *testing.T) {
	testOwnerManagement(t, "ChainOwner")
}

func TestNativeTokenOwnerManagement(t *testing.T) {
	testOwnerManagement(t, "NativeTokenOwner")
}

func testOwnerManagement(t *testing.T, ownerType string) {
	for _, version := range []uint64{params.ArbosVersion_51, params.ArbosVersion_60} {
		t.Run(fmt.Sprintf("ArbOS%d", version), func(t *testing.T) {
			testOwnerEventsForVersion(t, ownerType, version)
		})
	}
}

func testOwnerEventsForVersion(t *testing.T, ownerType string, arbosVersion uint64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arbOSInit := &params.ArbOSInit{
		NativeTokenSupplyManagementEnabled: true,
	}
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).WithArbOSVersion(arbosVersion).WithArbOSInit(arbOSInit)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// Get event topics
	arbOwnerABI, err := precompilesgen.ArbOwnerMetaData.GetAbi()
	Require(t, err)

	var addedTopic, removedTopic common.Hash
	var parseAdded, parseRemoved func(types.Log) (interface{}, error)

	if ownerType == "ChainOwner" {
		addedTopic = arbOwnerABI.Events["ChainOwnerAdded"].ID
		removedTopic = arbOwnerABI.Events["ChainOwnerRemoved"].ID
		parseAdded = func(log types.Log) (interface{}, error) {
			return arbOwner.ParseChainOwnerAdded(log)
		}
		parseRemoved = func(log types.Log) (interface{}, error) {
			return arbOwner.ParseChainOwnerRemoved(log)
		}
	} else {
		addedTopic = arbOwnerABI.Events["NativeTokenOwnerAdded"].ID
		removedTopic = arbOwnerABI.Events["NativeTokenOwnerRemoved"].ID
		parseAdded = func(log types.Log) (interface{}, error) {
			return arbOwner.ParseNativeTokenOwnerAdded(log)
		}
		parseRemoved = func(log types.Log) (interface{}, error) {
			return arbOwner.ParseNativeTokenOwnerRemoved(log)
		}
	}

	// Create test account
	builder.L2Info.GenerateAccount("TestOwner")
	testAddr := builder.L2Info.GetAddress("TestOwner")

	// 1. Check if account is NOT an owner
	isOwner, err := checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	Require(t, err)
	require.False(t, isOwner, "account should not be owner initially")

	// 2. Add account as owner
	tx, err := addOwner(arbOwner, &auth, ownerType, testAddr)
	Require(t, err)
	receipt, err := builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 3. Verify Added event was emitted (only for ArbOS 60)
	shouldEmit := arbosVersion >= params.ArbosVersion_60
	foundEvent := findAndParseEvent(receipt.Logs, addedTopic, parseAdded, testAddr)
	if shouldEmit {
		require.True(t, foundEvent, "%sAdded event should be emitted for ArbOS 60", ownerType)
	} else {
		require.False(t, foundEvent, "%sAdded event should NOT be emitted for ArbOS < 60", ownerType)
	}

	// 4. Check if account IS an owner
	isOwner, err = checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	Require(t, err)
	require.True(t, isOwner, "account should be owner after adding")

	// 5. Remove account as owner
	tx, err = removeOwner(arbOwner, &auth, ownerType, testAddr)
	Require(t, err)
	receipt, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// 6. Verify Removed event was emitted (only for ArbOS 60)
	foundEvent = findAndParseEvent(receipt.Logs, removedTopic, parseRemoved, testAddr)
	if shouldEmit {
		require.True(t, foundEvent, "%sRemoved event should be emitted for ArbOS 60", ownerType)
	} else {
		require.False(t, foundEvent, "%sRemoved event should NOT be emitted for ArbOS < 60", ownerType)
	}

	// 7. Check if account is NOT an owner
	isOwner, err = checkIsOwner(arbOwner, callOpts, ownerType, testAddr)
	Require(t, err)
	require.False(t, isOwner, "account should not be owner after removal")
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

func findAndParseEvent(logs []*types.Log, topic common.Hash, parseFunc func(types.Log) (interface{}, error), expectedAddr common.Address) bool {
	for _, lg := range logs {
		if lg.Topics[0] == topic {
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
	}
	return false
}
