// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// IDE entry points: one Test wrapper per scenario group.
package arbos

import (
	"testing"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestOwnership(t *testing.T) { systest.RunGroup(t, ownershipTests) }

func TestPrecompileInclusion(t *testing.T) { systest.RunGroup(t, precompileInclusionTests) }

func TestPrecompileSanity(t *testing.T) { systest.RunGroup(t, precompileSanityTests) }

func TestNativeToken(t *testing.T) { systest.RunGroup(t, nativeTokenTests) }
